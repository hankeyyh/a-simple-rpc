package client_discovery

import (
	"github.com/hankeyyh/a-simple-rpc/client"
)

// consul 服务发现
type ConsulDiscovery struct {
}

func (c *ConsulDiscovery) GetService() []*client.KVPair {
	//TODO implement me
	panic("implement me")
}

func (c *ConsulDiscovery) Close() {
	//TODO implement me
	panic("implement me")
}
