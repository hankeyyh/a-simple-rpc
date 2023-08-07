package client

import (
	"bufio"
	"context"
	"log"
	"net"
	"sync"
)

// 代表一次rpc调用
type Call struct {
	ServicePath   string
	ServiceMethod string
	Metadata      map[string]string
	ResMetadata   map[string]string
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call // invoke when call complete
}

func (c *Call) done() {
	select {
	case c.Done <- c:
		// ok
	default:
		log.Printf("rpc: discarding Call reply due to insufficient Done chan capacity")
	}
}

// 管理与服务实例之间的连接
type RPCClient interface {
	Connect(network, address string) error
	Go(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}, done chan *Call) *Call
	Call(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error
	Close() error

	IsClosing() bool
	IsShutdown() bool

	GetConn() net.Conn
}

type Client struct {
	option Option

	Conn net.Conn
	r    *bufio.Reader

	mutex    sync.Mutex
	seq      uint64
	pending  map[uint64]*Call
	closing  bool
	shutdown bool
}

func NewClient(option Option) *Client {
	client := &Client{
		option: option,
	}
	return client
}

func (c *Client) Connect(network, address string) error {
	return nil
}

func (c *Client) Go(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}, done chan *Call) *Call {
	//TODO implement me
	panic("implement me")
}

func (c *Client) GetConn() net.Conn {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Call(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Close() error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) IsClosing() bool {
	//TODO implement me
	panic("implement me")
}

func (c *Client) IsShutdown() bool {
	//TODO implement me
	panic("implement me")
}
