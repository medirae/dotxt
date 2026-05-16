package client

import (
	"context"
	"dotxt/config"
	"dotxt/rpc/shared"
	"dotxt/utils/testils"
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	testils.EnsureTestDir()
	path := "/tmp/dotxt-testing/rpc-client"
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

type Dummy struct{}

func (d *Dummy) Ping(args string, reply *string) error {
	*reply = "pong:" + args
	return nil
}

func startDummyUnixRPC(t *testing.T) (context.CancelFunc, *sync.WaitGroup) {
	t.Helper()

	l, err := net.Listen("unix", shared.Socket)
	require.NoError(t, err)

	srv := rpc.NewServer()
	require.NoError(t, srv.Register(&Dummy{}))

	ctx, ctxCancel := context.WithCancel(context.Background())
	cancel := func() {
		ctxCancel()
		l.Close()
	}
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		go func() {
			<-ctx.Done()
			if err := l.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				t.Logf("FAIL:: failed to close listener: %s", err)
			}
		}()
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go srv.ServeConn(conn)
		}
	}(ctx)

	return cancel, wg
}

func TestDial(t *testing.T) {
	assert := assert.New(t)

	cancel, wg := startDummyUnixRPC(t)
	client, err := Dial()
	require.NoError(t, err)

	var reply string
	require.NoError(t, client.Call("Dummy.Ping", "ping", &reply))
	assert.Equal("pong:ping", reply)

	client.Close()
	cancel()
	wg.Wait()
}
