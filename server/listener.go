package server

import "net"

type MakeListener func(address string) (net.Listener, error)

// 缓存ln
var makeListenerMap = make(map[string]MakeListener)

func init() {
	makeListenerMap["tcp"] = tcpMakeListener("tcp")
}

func tcpMakeListener(network string) MakeListener {
	return func(address string) (net.Listener, error) {
		return net.Listen(network, address)
	}
}
