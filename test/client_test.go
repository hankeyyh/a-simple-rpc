package test

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/client"
	"github.com/hankeyyh/a-simple-rpc/test/pb"
	"log"
	"sync"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	d, _ := client.NewPeer2PeerDiscovery("tcp@127.0.0.1:1234", "")
	option := client.DefaultOption
	option.Heartbeat = true
	option.HeartbeatInterval = time.Second
	xclient := pb.NewXClientForArith(client.FailFast, client.RandomSelect, d, option)
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

func TestMultiClient(t *testing.T) {
	d, _ := client.NewPeer2PeerDiscovery("tcp@127.0.0.1:1234", "")
	option := client.DefaultOption
	option.Heartbeat = true
	option.HeartbeatInterval = time.Second
	option.IdleTimeout = time.Second * 2
	xclient := pb.NewXClientForArith(client.FailTry, client.RandomSelect, d, option)
	arithClient := pb.NewArithClient(xclient)
	defer arithClient.Close()

	var a int32 = 2
	var b int32 = 3
	args := &pb.Args{
		A: &a,
		B: &b,
	}

	reply := &pb.Reply{}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		callMul(arithClient, args, reply, 1)
	}()
	go func() {
		callMul(arithClient, args, reply, 2)
	}()
	wg.Wait()
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
