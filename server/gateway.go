package server

import (
	"github.com/hankeyyh/a-simple-rpc/protocol"
	"github.com/soheilhy/cmux"
	"io"
	"log"
	"net"
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

}
