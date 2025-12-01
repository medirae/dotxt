package server

import (
	"context"
	"dotxt/logging"
	"dotxt/rpc/shared"
	"dotxt/terrors"
	"fmt"
	"strings"
	"sync"
	"time"
)

/*
here's the thing:
there should be a type Request that represents something that the server has to

	somehow respond to.
	- requests from the cli/gui:
		- read from todo data
		- write to todo data
		- read metadata
		- write metadata
	- requests from watcher:
		- false positive events that the application triggered i.e. writing to a todo or metadata change
		- write to a file by user
		- deletion of file
		- creation of file in directory
	- requests made from the server: ?

sources of incoming requests include the fsnotify.Watcher goroutine and service Methods.

	every source should wrap the received data into a Request and send it into a channel
	that solely receives requests; let's call the channel the requests-ch

there should be a goroutine reading Requests from requests-ch and categorizing them.
these categories must not have any collisions with eachother in terms of data corruption.

	but since that would be impossible, there needs to be a locking category that serves as a
	category that everything in it will lock every other category; it has the upper hand and
	blocks all else. so when something that collides with all or nearly all other category
	requests, must come here and block all as to avoid data corruption.
	but for any other category - presuming the locking category is not blocking - the Requests
	of each one can only block the requests of that category.
	when a requests could not possibly corrupt any data then that should go to a free-for-all kind
	of category where when a request comes it is immediately served.
	categories: ?

each category must have its own scheduler. the scheduler for the category

	must receive the nearly-immediately requests and go over them and *sort* and store them.
	then it must go over the list of Requests as long as they are non-blocking Requests and
	dispatch them. when the scheduler meets a blocking Request and all other non-blocking
	Requests, if any, are after that, then the scheduler locks everything down until that
	blocking Request is processed; after which it must unlock. when it unlocks it should check,
	if a significant portion of time has passed, it shouldn't resume the current list of Requests,
	but rather it should again check whether there are any newer Requests that have come during
	this significant period of time, add them to the list, sort them, and start going over them from the beginning.
*/

// Request Dispatcher listens to this for requests
var requestsCh chan *Request

// TODO: heavily review the concurrency management like done, ctx, etc
type Request struct {
	ID           string
	Resources    []shared.NamedLock
	Task         func(ctx context.Context) error
	Dependencies []<-chan struct{}
	done         chan struct{}
	ctx          context.Context
	CreationTime time.Time
}

func (r *Request) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("R:'%s'", r.ID))
	// if len(r.Resources) > 0 && r.Resources[0].Mutex == ExclusiveLock { // TODO: figure out how to mark exclusive lock
	// 	out.WriteString(" x")
	// }
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
func (r *Request) Exec(dispatcherCtx context.Context) error {
	defer close(r.done)
	for _, dependency := range r.Dependencies {
		select {
		case <-dependency:
		case <-r.ctx.Done():
			return fmt.Errorf("%w: %w: request '%s' cancelled via context: %w",
				terrors.ErrRPC, terrors.ErrConcurrency, r.ID, r.ctx.Err())
		case <-dispatcherCtx.Done():
			return fmt.Errorf("%w: %w: request '%s' cancelled via dispatcher: %w",
				terrors.ErrRPC, terrors.ErrConcurrency, r.ID, dispatcherCtx.Err())
		}
	}
	unlock, err := shared.AcquireLocks(r.Resources)
	if err != nil {
		return err
	}
	defer unlock()
	return r.Task(r.ctx)
}

func MakeRequest(
	label string, ctx context.Context,
	task func(ctx context.Context) error,
	resources []shared.NamedLock,
	deps ...<-chan struct{},
) (<-chan struct{}, context.CancelFunc, error) {
	rn := time.Now()
	id := fmt.Sprintf("%s:%d", strings.ReplaceAll(label, " ", "-"), rn.UnixMicro())
	ctxTimed, cancel := context.WithTimeout(ctx, requestTimeout)
	r := &Request{
		ID: id, Resources: resources, Task: task,
		Dependencies: deps, done: make(chan struct{}),
		ctx: ctxTimed, CreationTime: rn,
	}
	select {
	case requestsCh <- r:
	case <-time.After(requestAdmitTimeout):
		cancel()
		close(r.done)
		return nil, nil,
			fmt.Errorf("%w: request admittence timeout '%s' exceeded, server is busy, request dropped",
				terrors.ErrRPC, requestAdmitTimeout)
	}
	return r.done, cancel, nil
}

// reads requests and categorically decides where they ought to go
// must be run as a goroutine
/* TODO: bound concurrent processing
develop a `type Semaphore chan struct{}`
	with `func (s Semaphore) Acquire(ctx context.Context) error`
	and `func (s Semaphore) Release()`
then use Acquire before creating the request goroutine
	and then defer Release in the goroutine
ai said to set the size as min(4*CPU, 64) but I don't know what that means
*/
func Dispatcher(wg *sync.WaitGroup, ctx context.Context) {
	var requestGroup sync.WaitGroup
	defer requestGroup.Wait()
	defer wg.Done()

	type rh struct { // request holder
		r      *Request
		cancel context.CancelFunc
	}
	requests := make(map[string]rh)
	var requestsLock sync.RWMutex

	defer func() {
		requestsLock.Lock()
		for _, r := range requests {
			r.cancel()
		}
		requestsLock.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			logging.Logger.Infof("RPC server: Dispatcher: cancelled via main")
			return
		case r, ok := <-requestsCh:
			if !ok {
				logging.Logger.Infof("RPC server: Dispatcher: requests channel closed, exitting")
				return
			}

			requestGroup.Add(1)
			rctx, cancel := context.WithCancel(ctx)
			requestsLock.Lock()
			_, duplicate := requests[r.ID]
			if !duplicate {
				requests[r.ID] = rh{r: r, cancel: cancel}
			} else {
				cancel()
				close(r.done)
				requestsLock.Unlock()
				logging.Logger.Warnf("RPC server: Dispatcher: duplicate request '%s' ignored", *r)
				continue
			}
			requestsLock.Unlock()

			logging.Logger.Infof("RPC server: Dispatcher: goroutine started for request '%s'", *r)
			go func(r *Request) {
				defer requestGroup.Done()
				err := r.Exec(rctx)
				if err != nil {
					logging.Logger.Errorf("RPC server: Dispatcher: request goroutine: request '%s' Error: %w", *r, err)
				} else {
					logging.Logger.Infof("RPC server: Dispatcher: request goroutine: request '%s' finished", *r)
				}

				requestsLock.Lock()
				if val, ok := requests[r.ID]; ok {
					val.cancel()
					delete(requests, r.ID)
				}
				requestsLock.Unlock()
			}(r)
		}
	}
}
