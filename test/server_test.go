package test

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/server"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	"testing"
)

type Arith int

type Args struct {
	A int `json:"A"`
	B int `json:"B"`
}

type Reply struct {
	C int `json:"C"`
}

func (s *Arith) Add(ctx context.Context, args *proto.Args, reply *proto.Reply) error {
	c := args.GetA() + args.GetB()
	reply.C = &c
	return nil
}

func (s *Arith) Mul(ctx context.Context, args *proto.Args, reply *proto.Reply) error {
	c := args.GetA() * args.GetB()
	reply.C = &c
	return nil
}

func TestServer(t *testing.T) {
	svr := server.NewServer()
	defer svr.Close()

	svr.Register(new(Arith))
	svr.Serve("tcp", "127.0.0.1:1234")
}
