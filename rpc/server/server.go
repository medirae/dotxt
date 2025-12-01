package server

import (
	"context"
	"dotxt/logging"
	"dotxt/terrors"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	serverTimer         *time.Timer
	serverTimeout       time.Duration
	requestTimeout      time.Duration
	requestAdmitTimeout time.Duration
	requestChLoad       int
	// internalEventTimeout time.Duration
	externalEventTimeout time.Duration
)

func resetServerTimer() {
	serverTimer.Reset(serverTimeout)
}

func initSocket() string {
	socket := viper.GetString("rpc.socket") // TODO: (network) use port over localhost tcp
	if socket == "" {
		socket = "/tmp/dotxt.rpc" // TODO: (config) remove
	}
	os.Remove(socket) // TODO: check existence of previous server
	return socket
}

// TODO: (logging)

// TODO: implement a graceful shutdown
//  also listen to a variety of os signals

func RunServer() {
	var (
		listener net.Listener
		watcher  *fsnotify.Watcher
	)

	{
		// TODO: (config)
		requestsCh = make(chan *Request, requestChLoad)
	}
	{
		socket := initSocket()
		var err error
		listener, err = net.Listen("unix", socket)
		if err != nil {
			logging.Logger.Fatalf("%w: failed to listen: %w", terrors.ErrRPC, err)
		}
		defer listener.Close()
		logging.Logger.Infof("RPC server listening at '%s'", socket)
	}
	{
		// TODO: (config) internalEventTimeout
		// TODO: (config) also set requestTimeout
		// TODO: everytime a request comes, this has to be reset... or something like... I'd prefer to prevent churn
		serverTimeout = time.Duration(viper.GetInt64("rpc.server-timeout"))
		if serverTimeout == 0 { // TODO: (config) remove
			serverTimeout = 10
		}
		serverTimeout *= time.Second
		serverTimer = time.AfterFunc(serverTimeout, func() {
			logging.Logger.Info("shutting down after idle timout")
			// TODO: call graceful shutdown
		})
	}
	{
		var err error
		watcher, err = MakeWatcher()
		if err != nil {
			logging.Logger.Fatalf("%w: failed to watch: %w", terrors.ErrRPC, err)
		}
	}

	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background()) // TODO:
	{
		wg.Add(1)
		go Watch(&wg, ctx, watcher)
	}
	{
		wg.Add(1)
		go Listen(&wg, ctx, listener)
	}
	wg.Wait()
	cancel()
}

// uses rpc to listen to incoming requests
// must be run as a goroutine
func Listen(wg *sync.WaitGroup, done context.Context, listener net.Listener) {
	/*
		data:
			- check [<todolist>...]
				checkout the necesseties for timer management; maybe they should be requests too
			Write:
				- add <task> [--list=<todolist=todo>]
				- append <id> <task> [--list=<todolist=todo>]
				- deduplicate [--list==<todolist=todo>]
				- delete <id>... [--list==<todolist=todo>]
				- deprioritize <id>... [--list==<todolist=todo>]
				- done <id> [--list==<todolist=todo>]
				- increment id [val=1] [--list==<todolist=todo>]
				- migrate <from> [--list=<todolist=todo>]
				- move <from> <id> <to>
				- prepend <id> <task> [--list=<todolist=todo>]
				- prioritize <id> <priority> [--list=<todolist=todo>]
				- replace <id> <task> [--list=<todolist=todo>]
				- revert <id>... [--list==<todolist=todo>]
				- setc id val [--list==<todolist=todo>]
				- sort <todolist=todo>...
				- tc id [--list==<todolist=todo>]
			Read:
				- lsn id [--list==<todolist=todo>]
				- print <todolist=todo>...
				- print1 id [--list==<todolist=todo>]

		metadata:
			Read:
				- todo info
				- stats
				- config
			Write:
				- move/rename file
				- delete file
				- merge files
				- set config
	*/
	defer wg.Done()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return // listener closed
		}
		resetServerTimer()
		go rpc.ServeConn(conn) // manage goroutine
	}

}
