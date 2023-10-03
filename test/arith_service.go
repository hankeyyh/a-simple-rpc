package test

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/test/pb"
)

type ArithImp struct {
}

func (s *ArithImp) Sub(ctx context.Context, args *pb.Args, reply *pb.Reply) (err error) {
	c := args.GetA() - args.GetB()
	reply.C = &c
	return nil
}

func (s *ArithImp) Divide(ctx context.Context, args *pb.Args, reply *pb.Reply) (err error) {
	c := args.GetA() / args.GetB()
	reply.C = &c
	return nil
}

func (s *ArithImp) Add(ctx context.Context, args *pb.Args, reply *pb.Reply) error {
	c := args.GetA() + args.GetB()
	reply.C = &c
	return nil
}

func (s *ArithImp) Mul(ctx context.Context, args *pb.Args, reply *pb.Reply) error {
	c := args.GetA() * args.GetB()
	reply.C = &c
	return nil
}
