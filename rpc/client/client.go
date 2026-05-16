package client

import (
	"dotxt/rpc/shared"
	"net"
	"net/rpc"
	"time"
)

var (
	clientTimeout = 2 * time.Second
)

func Dial() (*rpc.Client, error) {
	conn, err := net.DialTimeout("unix", shared.Socket, clientTimeout)
	if err != nil {
		return nil, err
	}
	return rpc.NewClient(conn), nil
}
