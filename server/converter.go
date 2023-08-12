package server

import (
	"github.com/hankeyyh/a-simple-rpc/protocol"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

// http请求转为自定义rpc请求
func HttpRequest2RpcxRequest(r *http.Request) (*protocol.Message, error) {
	req := protocol.NewMessage()
	req.SetMessageType(protocol.Request)

	// seq
	h := r.Header
	seq := h.Get(XMessageID)
	if seq != "" {
		id, err := strconv.ParseUint(seq, 10, 64)
		if err != nil {
			return nil, err
		}
		req.SetSeq(id)
	}

	// heartbeat
	heartbeat := h.Get(XHeartbeat)
	if heartbeat != "" {
		req.SetHeartbeat(true)
	}

	// oneway
	oneway := h.Get(XOneway)
	if oneway != "" {
		req.SetOneway(true)
	}

	// serialize type
	serializeType := h.Get(XSerializeType)
	if serializeType != "" {
		st, err := strconv.Atoi(serializeType)
		if err != nil {
			return nil, err
		}
		req.SetSerializeType(protocol.SerializeType(st))
	}

	// metadata
	metadata := h.Get(XMeta)
	if metadata != "" {
		md, err := url.ParseQuery(metadata)
		if err != nil {
			return nil, err
		}
		mt := make(map[string]string)
		for k, v := range md {
			if len(v) > 0 {
				mt[k] = v[0]
			}
		}
		req.Metadata = mt
	}

	// service path & service method
	req.ServicePath = h.Get(XServicePath)
	req.ServiceMethod = h.Get(XServiceMethod)

	// payload
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	req.Payload = payload

	return req, nil
}
