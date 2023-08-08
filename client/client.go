package client

import (
	"bufio"
	"context"
	"log"
	"net"
	"sync"
	"time"
)

const (
	// ReaderBuffSize is used for bufio reader.
	ReaderBuffSize = 16 * 1024
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
	var conn net.Conn
	var err error

	switch network {
	case "http":
		// todo 建立http连接
	default:
		conn, err = newDirectConn(c, network, address)
	}

	if err != nil {
		return err
	}

	// 设置长连接
	if tc, ok := conn.(*net.TCPConn); ok && c.option.TCPKeepAlivePeriod > 0 {
		tc.SetKeepAlivePeriod(c.option.TCPKeepAlivePeriod)
		tc.SetKeepAlive(true)
	}

	// 设置超时时间
	if c.option.IdleTimeout != 0 {
		conn.SetDeadline(time.Now().Add(c.option.IdleTimeout))
	}

	c.Conn = conn
	c.r = bufio.NewReaderSize(conn, ReaderBuffSize)

	// 开启goroutine接收返回消息
	go c.input()

	// 心跳
	if c.option.Heartbeat && c.option.HeartbeatInterval > 0 {
		go c.heartbeat()
	}

	return nil
}

// 接收返回消息
func (c *Client) input() {
	var err error

	for err == nil {

	}
}

func (c *Client) heartbeat() {

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
