package consul_discovery

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/client"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	"log"
	"testing"
	"time"
)

func TestClientWithConsul(t *testing.T) {
	d, err := client.NewConsulDiscovery("Arith", "127.0.0.1:8500")
	if err != nil {
		t.Fatal(err)
	}
	option := client.DefaultOption
	xclient := client.NewXClient("Arith", client.FailOver, client.RandomSelect, d, option)
	defer xclient.Close()

	var a int32 = 2
	var b int32 = 3
	args := &proto.Args{
		A: &a,
		B: &b,
	}

	reply := &proto.Reply{}

	callMul(xclient, args, reply, 1)
}

func callMul(xclient client.XClient, args *proto.Args, reply *proto.Reply, i int) {
	for {
		err := xclient.Call(context.Background(), "Mul", args, reply)
		if err != nil {
			log.Fatalf("failed to call: %v", err)
		}

		log.Printf("[%d] %d * %d = %d", i, args.GetA(), args.GetB(), reply.GetC())
		time.Sleep(time.Second)
	}
}
