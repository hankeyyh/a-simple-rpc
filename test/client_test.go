package test

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/client"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	"log"
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
	err := xclient.Call(context.Background(), "Mul", args, reply)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	log.Printf("%d * %d = %d", args.GetA(), args.GetB(), reply.GetC())
	time.Sleep(1e9)

}
