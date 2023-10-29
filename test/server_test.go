package test

import (
	"github.com/hankeyyh/a-simple-rpc/server"
	"testing"
)

func TestServer(t *testing.T) {
	svr := server.NewServer()
	defer svr.Close()

	svr.RegisterName("Arith", new(ArithImp))
	svr.Serve("tcp", "127.0.0.1:1234")
}
