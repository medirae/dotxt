package server

import (
	"dotxt/rpc/shared"
	"fmt"
	"io/fs"
	"net"
	"net/rpc"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DummyService struct{}

func (s *DummyService) Pong(arg string, reply *string) error {
	*reply = fmt.Sprintf("%s:pong", arg)
	return nil
}

func init() {
	rpc.RegisterName("DummyService", &DummyService{})
}

// mimics client.Dial()
// copied here to avoid cycle imports
func ClientDial() (*rpc.Client, error) {
	clientTimeout := 2 * time.Second
	conn, err := net.DialTimeout("unix", shared.Socket, clientTimeout)
	if err != nil {
		return nil, err
	}
	return rpc.NewClient(conn), nil
}

func Ping(assert *assert.Assertions) error {
	cli, err := ClientDial()
	if err != nil {
		return err
	}
	defer cli.Close()

	reply := new(string)
	err = cli.Call("DummyService.Pong", "ping", reply)
	if err != nil {
		return err
	}
	assert.Equal("ping:pong", *reply)
	return nil
}

func TestRunServer(t *testing.T) {
	assert := assert.New(t)
	t.Run("normal", func(t *testing.T) {
		err, wg, cancel := RunServer()
		require.NoError(t, err)
		defer cancel()
		defer wg.Wait()
		err = Ping(assert)
		require.Nil(t, err)
	})
	t.Run("timeout", func(t *testing.T) {
		prevTO := serverTimeout
		defer func() { serverTimeout = prevTO }()
		serverTimeout = 2 * time.Second

		err, wg, _ := RunServer()
		require.NoError(t, err)
		err = Ping(assert)
		require.Nil(t, err)
		time.Sleep(serverTimeout + time.Second)
		wg.Wait()
		err = Ping(assert)
		require.Error(t, err)
		assert.ErrorIs(err, fs.ErrNotExist)
		var opErr *net.OpError
		assert.ErrorAs(err, &opErr)
		assert.Equal("dial", opErr.Op)
	})
}
