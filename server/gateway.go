package server

import (
	"context"
	"errors"
	"github.com/hankeyyh/a-simple-rpc/protocol"
	"github.com/hankeyyh/a-simple-rpc/share"
	"github.com/julienschmidt/httprouter"
	"github.com/soheilhy/cmux"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (svr *Server) startGateway(network string, ln net.Listener) net.Listener {
	// http请求/响应模型只能建立在tcp连接上
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		log.Printf("network is not tcp/tcp4/tcp6 so can not start gateway")
		return ln
	}

	m := cmux.New(ln)

	rpcxLn := m.Match(rpcxPrefixByteMatcher())

	// 开启gateway
	if !svr.DisableHTTPGateway {
		httpLn := m.Match(cmux.HTTP1Fast())
		go svr.startHTTP1APIGateway(httpLn)
	}

	go m.Serve()

	// rpc的请求交给自定义逻辑处理
	return rpcxLn
}

// 通过magicNumber判断是一个rpcx请求
func rpcxPrefixByteMatcher() cmux.Matcher {
	return func(reader io.Reader) bool {
		h := make([]byte, 1)
		n, _ := reader.Read(h)
		return n == 1 && h[0] == protocol.MagicNumber
	}
}

// 开启gateway
func (svr *Server) startHTTP1APIGateway(ln net.Listener) {
	router := httprouter.New()
	router.GET("/*servicePath", svr.handleGatewayRequest)
	router.POST("/*servicePath", svr.handleGatewayRequest)

	svr.mu.Lock()
	svr.gatewayHTTPServer = &http.Server{Handler: router}
	svr.mu.Unlock()

	if err := svr.gatewayHTTPServer.Serve(ln); err != nil {
		if errors.Is(err, ErrServerClosed) || errors.Is(err, cmux.ErrListenerClosed) || errors.Is(err, cmux.ErrServerClosed) {
			log.Print("gateway server closed")
		} else {
			log.Printf("error in gateway serve: %T %s", err, err)
		}
	}
}

// 处理http请求，并返回
func (svr *Server) handleGatewayRequest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// 缓存远端地址，这里RemoteAddr是string，而不是conn
	ctx := context.WithValue(context.Background(), RemoteConnContexKey, r.RemoteAddr)

	// 防止servicePath为空
	if r.Header.Get(XServicePath) == "" {
		servicePath := params.ByName("servicePath")
		servicePath = strings.TrimPrefix(servicePath, "/")
		r.Header.Set(XServicePath, servicePath)
	}
	servicePath := r.Header.Get(XServicePath)

	// http 请求转rpc请求
	reqMsg, err := HttpRequest2RpcxRequest(r)

	// response header
	wh := w.Header()
	wh.Set(XVersion, r.Header.Get(XVersion))
	wh.Set(XMessageID, r.Header.Get(XMessageID))
	if err == nil && servicePath == "" {
		err = errors.New("empty servicepath")
	} else {
		wh.Set(XServicePath, servicePath)
	}

	if err == nil && r.Header.Get(XServiceMethod) == "" {
		err = errors.New("empty servicemethod")
	} else {
		wh.Set(XServiceMethod, r.Header.Get(XServiceMethod))
	}

	if err == nil && r.Header.Get(XSerializeType) == "" {
		err = errors.New("empty serialized type")
	} else {
		wh.Set(XSerializeType, r.Header.Get(XSerializeType))
	}

	// 请求格式有问题
	if err != nil {
		rh := r.Header
		for k, v := range rh {
			if strings.HasPrefix(k, "X-SIMPLE-") && len(v) > 0 {
				wh.Set(k, v[0])
			}
		}
		wh.Set(XMessageStatusType, "Error")
		wh.Set(XErrorMessage, err.Error())
		w.WriteHeader(403)
		return
	}

	// ctx 缓存
	ctx = context.WithValue(ctx, StartRequestContextKey, time.Now().UnixNano())
	resMetadata := make(map[string]string)
	ctx = context.WithValue(ctx, share.ReqMetaDataKey, reqMsg.Metadata)
	ctx = context.WithValue(ctx, share.ResMetaDataKey, resMetadata)

	resMsg, err := svr.handleRequest(ctx, reqMsg)
	if err != nil {
		wh.Set(XMessageStatusType, "Error")
		wh.Set(XErrorMessage, err.Error())
		w.WriteHeader(500)
		return
	}

	// response metadata 写入header
	meta := url.Values{}
	for k, v := range resMsg.Metadata {
		meta.Add(k, v)
	}
	wh.Set(XMeta, meta.Encode())

	// payload 作为body返回，metadata，servicePath，serviceMethod已写在header中
	w.Write(resMsg.Payload)
}
