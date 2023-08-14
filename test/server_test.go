package test

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/server"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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

	var sd *protoreflect.ServiceDescriptor
	for i := 0; i < proto.File_arith_proto.Services().Len(); i++ {
		svc := proto.File_arith_proto.Services().Get(i)
		if svc.Name() == "Arith" {
			sd = &svc
			break
		}
	}

	svr.RegisterName("Arith", new(ArithImp), sd)
	svr.Serve("tcp", "127.0.0.1:1234")
}
