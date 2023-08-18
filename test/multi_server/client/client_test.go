package client

import (
	"context"
	"flag"
	"github.com/hankeyyh/a-simple-rpc/client"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	"log"
	"testing"
	"time"
)

var (
	addr1 = flag.String("addr1", "127.0.0.1:8972", "server1 address")
	addr2 = flag.String("addr2", "127.0.0.1:8973", "server2 address")
)

func TestClient(t *testing.T) {
	flag.Parse()

	d := client.NewMultiServersDiscovery([]*client.KVPair{{Key: *addr1}, {Key: *addr2}})
	option := client.DefaultOption
	option.Heartbeat = true
	option.HeartbeatInterval = time.Second
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
		err := xclient.Broadcast(context.Background(), "Mul", args, reply)
		if err != nil {
			log.Fatalf("failed to call: %v", err)
		}

		log.Printf("[%d] %d * %d = %d", i, args.GetA(), args.GetB(), reply.GetC())
		time.Sleep(time.Second)
	}
}
