package test

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/server"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	"testing"
)

type ArithImp struct {
}

func (s *ArithImp) Add(ctx context.Context, args *proto.Args, reply *proto.Reply) error {
	c := args.GetA() + args.GetB()
	reply.C = &c
	return nil
}

func (s *ArithImp) Mul(ctx context.Context, args *proto.Args, reply *proto.Reply) error {
	c := args.GetA() * args.GetB()
	reply.C = &c
	return nil
}

func TestServer(t *testing.T) {
	svr := server.NewServer()
	defer svr.Close()

	svr.RegisterName("arith", new(ArithImp))
	svr.Serve("tcp", "127.0.0.1:1234")
}
