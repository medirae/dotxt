package server

import (
	"dotxt/logging"
	"dotxt/rpc/shared"
	"dotxt/terrors"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
)

// TODO: listen to a variety of os signals (in main.go)

func RunServer() (error, *sync.WaitGroup, func() error) {
	var (
		listener net.Listener
		// watcher  *fsnotify.Watcher
		wg sync.WaitGroup
	)

	wg.Add(1)
	shutdown := func() error {
		os.Remove(shared.Socket)
		// wg.Done() TODO: uncomment for watcher
		return listener.Close()
	}

	{
		os.Remove(shared.Socket)
		var err error
		listener, err = net.Listen("unix", shared.Socket)
		if err != nil {
			return fmt.Errorf("%w: failed to listen: %w", terrors.ErrRPC, err), nil, nil
		}
		logging.Logger.Infof("RPC server listening at '%s'", shared.Socket)
	}
	serverTimer = time.AfterFunc(serverTimeout, func() {
		logging.Logger.Infof("shutting down after idle timout of %d", serverTimeout)
		if err := shutdown(); err != nil {
			logging.Logger.Warnf("RPC Server: RunServer: %w: failed to shut down server: %w", terrors.ErrRPC, err)
		}
	})

	{
		// var err error
		// watcher, err = MakeWatcher()
		// if err != nil {
		// 	logging.Logger.Fatalf("%w: failed to watch: %w", terrors.ErrRPC, err)
		// }
		// wg.Add(1)
		// go Watch(&wg, ctx, watcher)
	}

	go listen(&wg, listener)
	return nil, &wg, shutdown
}

// uses rpc to listen to incoming requests
// must be run as a goroutine
func listen(wg *sync.WaitGroup, listener net.Listener) {
	defer wg.Done()
	for {
		conn, err := listener.Accept()
		if err != nil {
			return // listener closed
		}
		serverTimer.Reset(serverTimeout)
		go rpc.ServeConn(conn) // TODO: investigate
	}
}
