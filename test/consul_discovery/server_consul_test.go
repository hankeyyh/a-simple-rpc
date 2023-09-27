package consul_discovery

import (
	"github.com/hankeyyh/a-simple-rpc/server"
	"github.com/hankeyyh/a-simple-rpc/server_plugin"
	"github.com/hankeyyh/a-simple-rpc/test"
	"testing"
	"time"
)

func TestServerWithConsul(t *testing.T) {
	svr := server.NewServer()
	defer svr.Close()

	consulPlugin := server_plugin.NewConsulPlugin(
		server_plugin.WithConsulAddr("127.0.0.1:8500"),
		server_plugin.WithServiceAddr("127.0.0.1:1234"),
		server_plugin.WithHeartbeat("192.168.65.254:1234", time.Second, 30*time.Second),
	)
	svr.Plugins.Add(consulPlugin)

	svr.RegisterName("Arith", new(test.ArithImp), "")
	svr.Serve("tcp", "127.0.0.1:1234")
}
