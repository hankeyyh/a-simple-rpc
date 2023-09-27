package test

import (
	"github.com/hankeyyh/a-simple-rpc/server"
	"github.com/hankeyyh/a-simple-rpc/server_plugin"
	"testing"
	"time"
)

func TestServerWithConsul(t *testing.T) {
	svr := server.NewServer()
	defer svr.Close()

	consulPlugin := server_plugin.NewConsulPlugin(
		server_plugin.WithConsulPath("127.0.0.1:8500"),
		server_plugin.WithServicePath("192.168.65.254:1234"),
		server_plugin.WithHeartbeat(time.Second, 30*time.Second),
	)
	svr.Plugins.Add(consulPlugin)

	svr.RegisterName("Arith", new(ArithImp), "")
	svr.Serve("tcp", "127.0.0.1:1234")
}
