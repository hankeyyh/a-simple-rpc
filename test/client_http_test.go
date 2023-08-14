package test

import (
	"bytes"
	"encoding/json"
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

type Args struct {
	A int `json:"A"`
	B int `json:"B"`
}

type Reply struct {
	C int `json:"C"`
}

func TestClientHttp(t *testing.T) {
	args := Args{
		A: 10,
		B: 20,
	}
	body, err := json.Marshal(args)
	if err != nil {
		log.Fatal("failed to json marshal: ", err)
	}
	req, err := http.NewRequest("GET", "http://127.0.0.1:1234", bytes.NewBuffer(body))
	if err != nil {
		log.Fatal("failed to create request: ", err)
	}
	h := req.Header
	h.Set(XMessageID, "10000")
	h.Set(XMessageType, "0")
	h.Set(XSerializeType, "2") // json
	h.Set(XServicePath, "Arith")
	h.Set(XServiceMethod, "Mul")
	h.Set("content-type", "application/json")

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
	reply := Reply{}
	err = json.Unmarshal(replyData, &reply)
	if err != nil {
		log.Fatal("failed to json unmarshal: ", err)
	}
	log.Printf("%d * %d = %d", args.A, args.B, reply.C)
}
