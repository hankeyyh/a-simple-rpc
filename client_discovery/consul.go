package client_discovery

import (
	"github.com/hankeyyh/a-simple-rpc/client"
	"sync"
)

// consul 服务发现
type ConsulDiscovery struct {
	// consul 地址
	ConsulPath []string
	// 服务地址
	ServicePath string

	//
	pairsMu sync.RWMutex
	pairs   []*client.KVPair
}

func (c *ConsulDiscovery) GetService() []*client.KVPair {
	c.pairsMu.RLock()
	defer c.pairsMu.RUnlock()
	return c.pairs
}

func (c *ConsulDiscovery) Close() {
	//TODO implement me
	panic("implement me")
}
