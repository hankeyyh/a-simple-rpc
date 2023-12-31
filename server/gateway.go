package server

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/hankeyyh/a-simple-rpc/protocol"
	"github.com/hankeyyh/a-simple-rpc/share"
	"github.com/iancoleman/strcase"
	"github.com/julienschmidt/httprouter"
	"github.com/soheilhy/cmux"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
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

// 健康检查
func (svr *Server) handleGatewayHealthCheck(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	w.WriteHeader(http.StatusOK)
	log.Printf("recv http heartbeat")
}

// 处理http请求，并返回
func (svr *Server) handleGatewayRequest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// 缓存远端地址，这里RemoteAddr是string，而不是conn
	ctx := context.WithValue(context.Background(), RemoteConnContexKey, r.RemoteAddr)
	var err error

	// 健康检查
	if r.URL.Path == "/health" {
		w.WriteHeader(http.StatusOK)
		log.Printf("recv http heartbeat")
		return
	}

	// 解析路径
	paths := strings.Split(r.URL.Path, "/")

	// response header
	wh := w.Header()
	if len(paths) > 3 {
		err = errors.New("path must be consist of service and method name")
		writeErrHeader(w, &wh, 403, err)
		return
	}
	// todo paths 与method.url 进行比较，method绑定的url可以自己设置
	servicePath := strcase.ToCamel(paths[1])
	serviceMethod := strcase.ToCamel(paths[2])

	if servicePath == "" {
		err = errors.New("empty servicepath")
	} else {
		r.Header.Set(XServicePath, servicePath)
		wh.Set(XServicePath, servicePath)
	}
	if serviceMethod == "" {
		err = errors.New("empty servicemethod")
	} else {
		r.Header.Set(XServiceMethod, serviceMethod)
		wh.Set(XServiceMethod, serviceMethod)
	}
	serializeType := r.Header.Get(XSerializeType)
	if serializeType == "" {
		err = errors.New("empty serialize type")
	} else {
		wh.Set(XSerializeType, serializeType)
	}
	wh.Set(XVersion, r.Header.Get(XVersion))
	wh.Set(XMessageID, r.Header.Get(XMessageID))
	wh.Set("content-type", "application/json")

	// request header中必填字段丢失
	if err != nil {
		writeErrHeader(w, &wh, 403, err)
		return
	}

	svc, err := svr.getService(servicePath)
	if err != nil {
		writeErrHeader(w, &wh, 403, err)
		return
	}
	mtype, ok := svc.methodMap[serviceMethod]
	if !ok {
		err = errors.New("can't find method " + serviceMethod)
		writeErrHeader(w, &wh, 403, err)
		return
	}

	// byte->json body/pb
	var argv = reflectTypePools.Get(mtype.ArgType)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrHeader(w, &wh, 403, err)
		return
	}
	if err = json.Unmarshal(body, &argv); err != nil {
		writeErrHeader(w, &wh, 403, err)
		return
	}
	// pb->byte
	st, _ := strconv.Atoi(serializeType)
	codec := share.Codecs[protocol.SerializeType(st)]
	payload, err := codec.Encode(argv)
	if err != nil {
		writeErrHeader(w, &wh, 403, err)
		return
	}

	// http 请求转rpc请求
	reqMsg, err := HttpRequest2RpcxRequest(r, payload)
	if err != nil {
		writeErrHeader(w, &wh, 403, err)
		return
	}

	// ctx 缓存
	ctx = context.WithValue(ctx, StartRequestContextKey, time.Now().UnixNano())
	resMetadata := make(map[string]string)
	ctx = context.WithValue(ctx, share.ReqMetaDataKey, reqMsg.Metadata)
	ctx = context.WithValue(ctx, share.ResMetaDataKey, resMetadata)

	// 处理请求
	resMsg, err := svr.handleRequest(ctx, reqMsg)
	if err != nil {
		writeErrHeader(w, &wh, 500, err)
		return
	}
	reflectTypePools.Put(mtype.ArgType, argv)

	// byte->pb/json body->byte
	replyv := reflectTypePools.Get(mtype.ReplyType)
	err = codec.Decode(resMsg.Payload, replyv)
	if err != nil {
		if replyv != nil {
			reflectTypePools.Put(mtype.ReplyType, replyv)
		}
		writeErrHeader(w, &wh, 500, err)
		return
	}
	body, err = json.Marshal(replyv)
	if err != nil {
		if replyv != nil {
			reflectTypePools.Put(mtype.ReplyType, replyv)
		}
		writeErrHeader(w, &wh, 500, err)
		return
	}
	if replyv != nil {
		reflectTypePools.Put(mtype.ReplyType, replyv)
	}

	// response metadata 写入header
	meta := url.Values{}
	for k, v := range resMsg.Metadata {
		meta.Add(k, v)
	}
	wh.Set(XMeta, meta.Encode())

	// 返回body，metadata，servicePath，serviceMethod已写在header中
	w.Write(body)
}

// header写入错误信息
func writeErrHeader(w http.ResponseWriter, wh *http.Header, code int, err error) {
	wh.Set(XMessageStatusType, "Error")
	wh.Set(XErrorMessage, err.Error())
	w.WriteHeader(code)
}
