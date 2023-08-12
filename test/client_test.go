package test

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/client"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	"log"
	"sync"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	d, _ := client.NewPeer2PeerDiscovery("tcp@127.0.0.1:1234", "")
	xclient := client.NewXClient("Arith", client.FailFast, client.RandomSelect, d, client.DefaultOption)
	defer xclient.Close()

	var a int32 = 2
	var b int32 = 3
	args := &proto.Args{
		A: &a,
		B: &b,
	}

	reply := &proto.Reply{}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		callMul(xclient, args, reply, 1)
	}()
	go func() {
		callMul(xclient, args, reply, 2)
	}()
	wg.Wait()
}

func callMul(xclient client.XClient, args *proto.Args, reply *proto.Reply, i int) {
	for {
		err := xclient.Call(context.Background(), "Mul", args, reply)
		if err != nil {
			log.Fatalf("failed to call: %v", err)
		}

		log.Printf("[%d] %d * %d = %d", i, args.GetA(), args.GetB(), reply.GetC())
		time.Sleep(1e9)
	}
}
