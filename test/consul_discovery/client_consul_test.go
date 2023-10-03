package consul_discovery

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/client"
	"github.com/hankeyyh/a-simple-rpc/test/pb"
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
	xclient := pb.NewXClientForArith("Arith", client.FailOver, client.RandomSelect, d, option)
	arithClient := pb.NewArithClient(xclient)
	defer arithClient.Close()

	var a int32 = 2
	var b int32 = 3
	args := &pb.Args{
		A: &a,
		B: &b,
	}

	reply := &pb.Reply{}

	callMul(arithClient, args, reply, 1)
}

func callMul(arithClient *pb.ArithClient, args *pb.Args, reply *pb.Reply, i int) {
	var err error
	for {
		reply, err = arithClient.Mul(context.Background(), args)
		if err != nil {
			log.Fatalf("failed to call: %v", err)
		}

		log.Printf("[%d] %d * %d = %d", i, args.GetA(), args.GetB(), reply.GetC())
		time.Sleep(time.Second)
	}
}
