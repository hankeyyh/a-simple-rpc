package test

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
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
