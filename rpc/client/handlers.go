package client

import (
	"dotxt/logging"
	"dotxt/rpc/server"
	"dotxt/terrors"
	"fmt"
	"net/rpc"
)

func callFunc(cli *rpc.Client, uri string, args any, reply any) {
	if err := cli.Call(uri, args, reply); err != nil {
		logging.Logger.Fatal(fmt.Errorf("%w: %w", terrors.ErrRPC, err))
	}
}

func AddTask(task, path string) error {
	cli, err := Dial()
	if err != nil {
		logging.Logger.Fatal(fmt.Errorf("%w: client could not connect to server: %w", terrors.ErrRPC, err))
	}
	defer cli.Close()

	var reply *server.ArgErr
	callFunc(cli, "TaskService.AddTask", &server.ArgTask{Task: task, Path: path}, &reply)
	return reply.Error()
}
