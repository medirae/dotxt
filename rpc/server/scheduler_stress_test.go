package server

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func initDispatcher(t *testing.T) (context.CancelFunc, *sync.WaitGroup) {
	t.Helper()

	numWorkers = 16
	requestTimeout = 1 * time.Second
	requestAdmitTimeout = 200 * time.Millisecond
	requestsCh = make(chan *RequestHandle, 16)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go Dispatcher(&wg, ctx)

	return cancel, &wg
}

func offDispatcher(cancel context.CancelFunc, wg *sync.WaitGroup) {
	cancel()
	wg.Wait()
}

// can the scheduler survive a flood
func TestThroughputStress(t *testing.T) {
	cancel, wg := initDispatcher(t)
	defer offDispatcher(cancel, wg)

	const total = 50_000

	var completed atomic.Int64

	for cnt := range total {
		_, err := MakeRequest(
			fmt.Sprintf("throughput-%d", cnt),
			func(ctx context.Context) error {
				completed.Add(1)
				return nil
			},
			nil,
		)
		require.NoError(t, err)
	}

	deadline := time.After(10 * time.Second)
	for {
		if completed.Load() == total {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout: completed=%d/%d", completed.Load(), total)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}
