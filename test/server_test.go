package test

import (
	"context"
	"fmt"
	"github.com/hankeyyh/a-simple-rpc/server"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	"testing"
	"time"
)

type Arith int

type Args struct {
	A int
	B int
}

type Reply struct {
	C int
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
	svr.Register(new(Arith))
	svr.Serve("tcp", "127.0.0.1:1234")
}

func Test1(t *testing.T) {
	var c = make(chan int, 10)

	go func() {
		for {
			select {
			case a := <-c:
				fmt.Println("a: ", a)
			case b := <-c:
				fmt.Println("b ", b)
			}
		}
	}()

	for { // send random sequence of bits to c
		select {
		case c <- 0: // note: no statement, no fallthrough, no folding of cases
			fmt.Println("input 0")
		case c <- 1:
			fmt.Println("input 1")
		}
		time.Sleep(time.Second)
	}

	select {} // block forever
}
