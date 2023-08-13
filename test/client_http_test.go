package test

import (
	"bytes"
	"github.com/hankeyyh/a-simple-rpc/codec"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	"io"
	"log"
	"net/http"
	"testing"
)

const (
	XVersion           = "X-SIMPLE-Version"
	XMessageType       = "X-SIMPLE-MessageType"
	XHeartbeat         = "X-SIMPLE-Heartbeat"
	XOneway            = "X-SIMPLE-Oneway"
	XMessageStatusType = "X-SIMPLE-MessageStatusType"
	XSerializeType     = "X-SIMPLE-SerializeType"
	XMessageID         = "X-SIMPLE-MessageID"
	XServicePath       = "X-SIMPLE-ServicePath"
	XServiceMethod     = "X-SIMPLE-ServiceMethod"
	XMeta              = "X-SIMPLE-Meta"
	XErrorMessage      = "X-SIMPLE-ErrorMessage"
)

func TestClientHttp(t *testing.T) {
	cc := &codec.PBCodec{}

	var a int32 = 10
	var b int32 = 20

	args := &proto.Args{
		A: &a,
		B: &b,
	}
	data, _ := cc.Encode(args)

	req, err := http.NewRequest("GET", "http://127.0.0.1:1234", bytes.NewBuffer(data))
	if err != nil {
		log.Fatal("failed to create request: ", err)
	}
	h := req.Header
	h.Set(XMessageID, "10000")
	h.Set(XMessageType, "0")
	h.Set(XSerializeType, "2") // json
	h.Set(XServicePath, "Arith")
	h.Set(XServiceMethod, "Mul")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("failed to call: ", err)
	}
	defer res.Body.Close()

	replyData, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("failed to read response: ", err)
	} else if len(replyData) == 0 {
		log.Fatal("reply is empty")
	}
	reply := proto.Reply{}
	err = cc.Decode(replyData, &reply)
	if err != nil {
		log.Fatal("failed to decode reply: ", err)
	}
	log.Printf("%d * %d = %d", *args.A, *args.B, *reply.C)
}
