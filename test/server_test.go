package test

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/server"
	"testing"
)

type Arith int

type Args struct {
	A int
	B int
}

type Reply struct {
	C int
}

func (s *Arith) Add(ctx context.Context, args *Args, reply *Reply) error {
	reply.C = args.A + args.B
	return nil
}

func TestServer(t *testing.T) {
	svr := server.NewServer()
	svr.Register(new(Arith))
	svr.Serve("tcp", "127.0.0.1:1234")
}
