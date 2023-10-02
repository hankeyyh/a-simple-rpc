package client

import (
	"errors"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"sync"
	"time"
)

var (
	ErrServiceNotFound = errors.New("service not found from consul agent")
)

// consul 服务发现
type ConsulDiscovery struct {
	// consul 地址
	ConsulAddr string
	// 服务
	ServiceName string

	// 服务地址
	pairsMu sync.RWMutex
	pairs   []*KVPair

	mu    sync.Mutex
	chans []chan []*KVPair // 每个客户端一个chan用于接收改变信号
}

func NewConsulDiscovery(serviceName string, consulAddr string) (*ConsulDiscovery, error) {
	cd := new(ConsulDiscovery)
	cd.ConsulAddr = consulAddr
	cd.ServiceName = serviceName

	pairs, err := cd.findServiceAddress()
	if err != nil {
		fmt.Printf("findServiceAddress fail, err: %v\n", err)
		return nil, ErrServiceNotFound
	}
	cd.pairsMu.Lock()
	cd.pairs = pairs
	cd.pairsMu.Unlock()

	// 监控服务变化
	go cd.watch()

	return cd, nil
}

func (c *ConsulDiscovery) GetService() []*KVPair {
	c.pairsMu.RLock()
	defer c.pairsMu.RUnlock()

	return c.pairs
}

// 返回chan用于接收服务变化信号
func (c *ConsulDiscovery) WatchService() chan []*KVPair {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan []*KVPair, 10) // 给一个缓冲，chan中最多可以放10次改变的地址，防止客户端没有及时处理
	c.chans = append(c.chans, ch)  // 一个ch对应一个客户端，当服务变化，需要通知所有xclient
	return ch
}

func (c *ConsulDiscovery) RemoveWatcher(ch chan []*KVPair) {
	c.mu.Lock()
	defer c.mu.Unlock()

	chans := make([]chan []*KVPair, 0)
	for _, cc := range c.chans {
		if cc == ch {
			continue
		}
		chans = append(chans, cc)
	}
	c.chans = chans
}

func (c *ConsulDiscovery) findServiceAddress() ([]*KVPair, error) {
	config := api.DefaultConfig()
	config.Address = c.ConsulAddr

	cclient, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	svcMap, err := cclient.Agent().ServicesWithFilter(fmt.Sprintf("Service == \"%s\"", c.ServiceName))
	if err != nil {
		return nil, err
	}

	pairs := make([]*KVPair, 0)
	for _, v := range svcMap {
		addr := fmt.Sprintf("%s:%d", v.Address, v.Port)
		pairs = append(pairs, &KVPair{
			Key:   addr,
			Value: "",
		})
	}
	return pairs, nil
}

func (c *ConsulDiscovery) Close() {

}

func (c *ConsulDiscovery) watch() {
	params := make(map[string]interface{})
	params["type"] = "service"
	params["service"] = c.ServiceName

	plan, err := watch.Parse(params)
	if err != nil {
		fmt.Println("watch.Parse fail, err: ", err)
		return
	}

	plan.Handler = func(u uint64, raw interface{}) {
		entries := raw.([]*api.ServiceEntry)
		pairs := make([]*KVPair, 0, len(entries))
		for _, entry := range entries {
			fmt.Printf("service %s is available at %s:%d, id: %s\n", entry.Service.Service, entry.Service.Address,
				entry.Service.Port, entry.Service.ID)
			pairs = append(pairs, &KVPair{
				Key:   fmt.Sprintf("%s:%d", entry.Service.Address, entry.Service.Port),
				Value: "",
			})
		}

		// 更新pairs
		c.pairsMu.Lock()
		c.pairs = pairs
		c.pairsMu.Unlock()

		// 通知所有客户端，服务地址改变了
		c.mu.Lock()
		for _, ch := range c.chans {
			ch := ch
			go func() {
				defer func() {
					recover()
				}()
				select {
				case ch <- c.pairs:
				case <-time.After(time.Minute):
					fmt.Println("chan is full and new change has been dropped")
				}
			}()
		}
		c.mu.Unlock()
	}

	if err = plan.Run(c.ConsulAddr); err != nil {
		fmt.Println("plan.Run fail, err: ", err)
		return
	}
}
