package server_plugin

import (
	"fmt"
	consul_api "github.com/hashicorp/consul/api"
	"net"
	"strconv"
	"time"
)

// consul服务发现对应服务端的plugin
type ConsulPlugin struct {
	// consul 地址
	ConsulAddr string
	// 服务地址
	ServiceAddr string
	ServiceName string
	// 健康检查
	HeartbeatAddr       string
	HeartbeatInternal   time.Duration
	MaxWaitForHeartbeat time.Duration
}

type ConsulOpt func(plugin *ConsulPlugin)

func WithConsulAddr(addr string) ConsulOpt {
	return func(c *ConsulPlugin) {
		c.ConsulAddr = addr
	}
}

func WithServiceAddr(addr string) ConsulOpt {
	return func(c *ConsulPlugin) {
		c.ServiceAddr = addr
	}
}

func WithHeartbeat(heartbeatAddr string, interval, maxWait time.Duration) ConsulOpt {
	return func(c *ConsulPlugin) {
		c.HeartbeatAddr = heartbeatAddr
		c.HeartbeatInternal = interval
		c.MaxWaitForHeartbeat = maxWait
	}
}

func NewConsulPlugin(opts ...ConsulOpt) *ConsulPlugin {
	consulPlugin := new(ConsulPlugin)
	for _, opt := range opts {
		opt(consulPlugin)
	}
	return consulPlugin
}

func (c *ConsulPlugin) Register(name string, rcvr interface{}, metadata string) error {
	c.ServiceName = name

	// consul 配置
	config := consul_api.DefaultConfig()
	config.Address = c.ConsulAddr

	cclient, err := consul_api.NewClient(config)
	if err != nil {
		return err
	}
	// 注册配置项
	svrHost, port, err := net.SplitHostPort(c.ServiceAddr)
	if err != nil {
		return err
	}
	svrPort, err := strconv.Atoi(port)
	if err != nil {
		return err
	}
	httpPath := fmt.Sprintf("http://%s/health", c.HeartbeatAddr)
	registry := new(consul_api.AgentServiceRegistration)
	registry.Name = name
	registry.ID = fmt.Sprintf("%s-%s:%d", name, svrHost, svrPort)
	registry.Address = svrHost
	registry.Port = svrPort
	registry.Check = &consul_api.AgentServiceCheck{
		HTTP:                           httpPath,
		TCPUseTLS:                      false,
		TLSSkipVerify:                  true,
		Interval:                       c.HeartbeatInternal.String(),
		DeregisterCriticalServiceAfter: c.MaxWaitForHeartbeat.String(),
	}
	// 服务注册
	err = cclient.Agent().ServiceRegister(registry)
	return err
}

func (c *ConsulPlugin) UnRegister(name string) error {
	config := consul_api.DefaultConfig()
	config.Address = c.ConsulAddr
	cclient, err := consul_api.NewClient(config)
	if err != nil {
		return err
	}
	err = cclient.Agent().ServiceDeregister(c.ServiceName)
	return err
}

func (c *ConsulPlugin) Start() error {
	return nil
}

func (c *ConsulPlugin) Stop() error {
	return nil
}
