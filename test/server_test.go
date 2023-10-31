package test

import (
	"github.com/hankeyyh/a-simple-rpc/server"
	"github.com/hankeyyh/a-simple-rpc/test/pb"
	"testing"
)

func TestServer(t *testing.T) {
	svr := server.NewServer()
	defer svr.Close()

	//svr.RegisterName("Arith", new(ArithImp))
	err := pb.RegisterArithServiceByProto(svr, new(ArithImp))
	if err != nil {
		panic(err)
	}
	svr.Serve("tcp", "127.0.0.1:1234")
}
