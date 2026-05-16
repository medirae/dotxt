package server

import (
	"context"
	"dotxt/config"
	"dotxt/terrors"
	"dotxt/utils/testils"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	testils.EnsureTestDir()
	path := "/tmp/dotxt-testing/rpc-server"
	if err := os.RemoveAll(path); err != nil {
		panic(err)
	}
	config.InitViper(path)
	if err := os.MkdirAll(path, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// helper to start a dispatcher and return cancel+wait cleanup
func startDispatcher(t *testing.T) (cancel context.CancelFunc, wg *sync.WaitGroup) {
	t.Helper()

	numWorkers = 2
	requestTimeout = 0
	requestAdmitTimeout = 200 * time.Millisecond
	requestsCh = make(chan *RequestHandle, 16)

	wg = &sync.WaitGroup{}
	wg.Add(1)
	ctx, c := context.WithCancel(context.Background())
	go Dispatcher(wg, ctx)

	return c, wg
}

func TestDispatcher(t *testing.T) {
	assert := assert.New(t)
	t.Run("process request", func(t *testing.T) {
		cancel, wg := startDispatcher(t)
		defer func() {
			cancel()
			wg.Wait()
		}()

		doneCh := make(chan struct{}, 1)
		rh, err := MakeRequest("process-test", func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			doneCh <- struct{}{}
			return nil
		}, nil)
		require.NoError(t, err)
		require.NoError(t, rh.Wait())
		select {
		case <-doneCh:
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("task body did not run")
		}
	})
	t.Run("rejects duplicate", func(t *testing.T) {
		cancel, wg := startDispatcher(t)
		defer func() {
			cancel()
			wg.Wait()
		}()

		r := &Request{
			ID:   "dup-id",
			Task: func(ctx context.Context) error { time.Sleep(30 * time.Millisecond); return nil },
		}
		rh1 := newRequestHandle(r, context.Background(), 0)
		rh2 := newRequestHandle(r, context.Background(), 0)

		select {
		case requestsCh <- rh1:
		default:
			t.Fatalf("failed to enqueue first request")
		}

		time.Sleep(5 * time.Millisecond) // pause to allow registration
		select {
		case requestsCh <- rh2:
		default:
			t.Fatalf("failed to enqueue duplicate request")
		}

		err := rh2.Wait()
		require.Error(t, err)
		assert.ErrorIs(err, terrors.ErrConcurrency)
		assert.ErrorIs(err, terrors.ErrRPC)
		assert.ErrorContains(err, "duplicate")

		require.NoError(t, rh1.Wait())
	})
	t.Run("shutdown cancels pending", func(t *testing.T) {
		// to increase chance of queueing
		numWorkers = 1
		requestTimeout = 0
		requestAdmitTimeout = 200 * time.Millisecond
		requestsCh = make(chan *RequestHandle, 4)

		wg := &sync.WaitGroup{}
		wg.Add(1)
		ctx, cancel := context.WithCancel(context.Background())
		go Dispatcher(wg, ctx)

		r := &Request{
			ID: "long-job",
			Task: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(5 * time.Second):
					return nil
				}
			},
		}
		rh := newRequestHandle(r, context.Background(), 0)
		requestsCh <- rh
		time.Sleep(5 * time.Millisecond) // let dispatcher pick it up

		cancel()
		wg.Wait()
		err := rh.Wait()
		require.Error(t, err)
		assert.ErrorIs(err, context.Canceled)
	})
}
