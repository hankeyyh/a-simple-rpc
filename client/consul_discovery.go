package client

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"sync"
)

// consul 服务发现
type ConsulDiscovery struct {
	// consul 地址
	ConsulAddr string
	// 服务
	ServiceName string

	//
	pairsMu sync.RWMutex
	pairs   []*KVPair
}

func NewConsulDiscovery(serviceName string, consulAddr string) *ConsulDiscovery {
	cd := new(ConsulDiscovery)
	cd.ConsulAddr = consulAddr
	cd.ServiceName = serviceName
	return cd
}

func (c *ConsulDiscovery) GetService() []*KVPair {
	config := api.DefaultConfig()
	config.Address = c.ConsulAddr

	cclient, err := api.NewClient(config)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	svcMap, err := cclient.Agent().ServicesWithFilter(fmt.Sprintf("Service == \"%s\"", c.ServiceName))
	if err != nil {
		fmt.Println(err)
		return nil
	}

	c.pairsMu.RLock()
	defer c.pairsMu.RUnlock()

	for _, v := range svcMap {
		addr := fmt.Sprintf("%s:%d", v.Address, v.Port)
		c.pairs = append(c.pairs, &KVPair{
			Key:   addr,
			Value: "",
		})
	}

	return c.pairs
}

func (c *ConsulDiscovery) Close() {

}
