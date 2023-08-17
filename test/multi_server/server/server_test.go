package server

import (
	"context"
	"flag"
	"github.com/hankeyyh/a-simple-rpc/server"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	"log"
	"testing"
)

var (
	addr1 = flag.String("addr1", "127.0.0.1:8972", "server1 address")
	addr2 = flag.String("addr2", "127.0.0.1:8973", "server2 address")
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
	flag.Parse()

	go createServer(*addr1)
	go createServer(*addr2)

	select {}
}

func createServer(addr string) {
	svr := server.NewServer()
	defer svr.Close()

	svr.RegisterName("Arith", new(ArithImp))
	log.Println("server start at: ", addr)
	err := svr.Serve("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
}
