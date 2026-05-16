package client

import (
<<<<<<< Updated upstream
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
||||||| Stash base
=======
	"net"
	"net/rpc"
	"time"

	"github.com/spf13/viper"
)

func Dial() (*rpc.Client, error) {
	socket := viper.GetString("rpc.socket")
	if socket == "" { // TODO: (config) remove
		socket = "/tmp/dotxt.rpc"
	}

	timeout := time.Duration(viper.GetInt64("rpc.client-timeout"))
	if timeout == 0 { // TODO: (config) remove
		timeout = 2
	}
	timeout *= time.Second

	conn, err := net.DialTimeout("unix", socket, timeout)
>>>>>>> Stashed changes
	if err != nil {
		return nil, err
	}
	return rpc.NewClient(conn), nil
}
