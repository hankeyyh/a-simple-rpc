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

type ConsulDiscoveryOption func(c *ConsulDiscovery)

func NewConsulDiscovery(options ...ConsulDiscoveryOption) *ConsulDiscovery {
	cd := new(ConsulDiscovery)

	for _, option := range options {
		option(cd)
	}
	return cd
}

func WithConsulPath(path []string) ConsulDiscoveryOption {
	return func(c *ConsulDiscovery) {
		c.ConsulPath = path
	}
}

func WithServicePath(path string) ConsulDiscoveryOption {
	return func(c *ConsulDiscovery) {
		c.ServicePath = path
	}
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
