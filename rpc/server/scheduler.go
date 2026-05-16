package server

import (
	"context"
	"dotxt/logging"
	"dotxt/rpc/shared"
	"dotxt/terrors"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type RequestHandle struct {
	request *Request
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	mu      sync.Mutex
	err     error
}

func (rh *RequestHandle) String() string { return rh.request.String() }

func newRequestHandle(r *Request, ctx context.Context, timeout time.Duration) *RequestHandle {
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	return &RequestHandle{request: r, ctx: ctx, cancel: cancel, done: make(chan struct{})}
}

func (rh *RequestHandle) Cancel() { rh.cancel() }

func (rh *RequestHandle) Done() <-chan struct{} { return rh.done }

func (rh *RequestHandle) Wait() error {
	<-rh.done
	rh.mu.Lock()
	defer rh.mu.Unlock()
	return rh.err
}

func (rh *RequestHandle) setResult(err error) {
	rh.mu.Lock()
	if rh.err != nil {
		rh.err = errors.Join(rh.err, err)
	} else {
		rh.err = err
	}
	rh.mu.Unlock()

	// ensure done is closed without panic
	select {
	case <-rh.done:
	default:
		close(rh.done)
	}
}

type Request struct {
	ID           string
	Resources    []shared.NamedLock
	Task         func(ctx context.Context) error
	Dependencies []<-chan struct{}
	CreationTime time.Time
}

func (r *Request) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("R:'%s'", r.ID))
	if len(r.Resources) > 0 {
		out.WriteString(fmt.Sprintf(" %dr", len(r.Resources)))
	}
	if len(r.Dependencies) > 0 {
		out.WriteString(fmt.Sprintf(" %dd", len(r.Dependencies)))
	}
	out.WriteString(r.CreationTime.Format("@2006-Jan-02--03-04-05--Z0700"))
	return out.String()
}

// executing the request along with pre and post processing
// must be run as a goroutine
func (r *Request) Exec(ctx context.Context, done chan<- struct{}) error {
	defer close(done)
	for _, dependency := range r.Dependencies { // WARN: prone to deadlock
		select {
		case <-dependency:
		case <-ctx.Done():
			return fmt.Errorf("%w: %w: request '%s' cancelled: %w",
				terrors.ErrRPC, terrors.ErrConcurrency, r.ID, ctx.Err())
		}
	}
	unlock, err := shared.AcquireLocks(r.Resources)
	if err != nil {
		return err
	}
	tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	defer unlock()
	return r.Task(tctx)
}

func MakeRequest(
	label string, task func(ctx context.Context) error,
	resources []shared.NamedLock,
	deps ...<-chan struct{},
) (*RequestHandle, error) {
	rn := time.Now()
	id := fmt.Sprintf("%s:%d", strings.ReplaceAll(label, " ", "-"), rn.UnixMicro())
	r := &Request{
		ID: id, Resources: resources, Task: task,
		Dependencies: deps, CreationTime: rn,
	}
	rh := newRequestHandle(r, context.Background(), requestTimeout)
	select {
	case requestsCh <- rh:
	case <-time.After(requestAdmitTimeout):
		rh.Cancel()
		return nil, fmt.Errorf("%w: request admittence timeout '%s' exceeded, server is busy, request dropped",
			terrors.ErrRPC, requestAdmitTimeout)
	}
	return rh, nil
}

// reads requests and categorically decides where they ought to go
// must be run as a goroutine
func Dispatcher(wg *sync.WaitGroup, ctx context.Context) {
	var requestGroup sync.WaitGroup
	type rho struct { // request handle holder
		rh     *RequestHandle
		cancel context.CancelFunc
	}
	requests := make(map[string]rho)
	var requestsLock sync.RWMutex

	var workerGroup sync.WaitGroup
	jobs := make(chan func(), numWorkers)
	for range numWorkers {
		workerGroup.Add(1)
		go func() {
			defer workerGroup.Done()
			for job := range jobs {
				job()
			}
		}()
	}

	// shutdown
	defer func() {
		close(jobs)
		requestsLock.Lock()
		for _, r := range requests {
			r.cancel()
			r.rh.setResult(fmt.Errorf("RPC server: Dispatcher: %w: %w: shutting down", terrors.ErrRPC, terrors.ErrConcurrency))
		}
		requestsLock.Unlock()
		requestGroup.Wait()
		workerGroup.Wait()
		wg.Done()
	}()

	// eventloop
	for {
		select {
		case <-ctx.Done():
			logging.Logger.Infof("RPC server: Dispatcher: cancelled via main")
			return
		case rh, ok := <-requestsCh:
			if !ok {
				logging.Logger.Infof("RPC server: Dispatcher: requests channel closed, exitting")
				return
			}

			requestGroup.Add(1)
			rctx, cancel := context.WithCancel(ctx)
			requestsLock.Lock()
			_, duplicate := requests[rh.request.ID]
			if !duplicate {
				requests[rh.request.ID] = rho{rh: rh, cancel: cancel}
			} else {
				cancel()
				requestsLock.Unlock()
				rh.setResult(fmt.Errorf("RPC server: %w: %w: duplicate request '%s' ignored", terrors.ErrRPC, terrors.ErrConcurrency, rh.request.ID))
				logging.Logger.Warnf("RPC server: Dispatcher: duplicate request '%s' ignored", rh.String())
				requestGroup.Done()
				continue
			}
			requestsLock.Unlock()

			logging.Logger.Infof("RPC server: Dispatcher: goroutine started for request '%s'", rh.String())
			jobs <- func() {
				defer requestGroup.Done()
				err := rh.request.Exec(rctx, rh.done)
				rh.setResult(err)
				if err != nil {
					logging.Logger.Errorf("RPC server: Dispatcher: request goroutine: request '%s' Error: %w", *rh.request, err)
				} else {
					logging.Logger.Infof("RPC server: Dispatcher: request goroutine: request '%s' finished", *rh.request)
				}

				requestsLock.Lock()
				if val, ok := requests[rh.request.ID]; ok {
					val.cancel()
					delete(requests, rh.request.ID)
				}
				requestsLock.Unlock()
			}
		}
	}
}
