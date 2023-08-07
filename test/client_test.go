package test

import (
	"context"
	"github.com/hankeyyh/a-simple-rpc/client"
	"log"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	d, _ := client.NewPeer2PeerDiscovery("tcp@127.0.0.1:1234", "")
	xclient := client.NewXClient("Arith", client.FailTry, client.RandomSelect, d, client.DefaultOption)
	defer xclient.Close()

	args := &Args{
		A: 10,
		B: 20,
	}

	for {
		reply := &Reply{}
		err := xclient.Call(context.Background(), "Mul", args, reply)
		if err != nil {
			log.Fatalf("failed to call: %v", err)
		}

		log.Printf("%d * %d = %d", args.A, args.B, reply.C)
		time.Sleep(1e9)
	}
}
