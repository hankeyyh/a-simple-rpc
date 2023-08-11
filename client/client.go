package client

import (
	"bufio"
	"context"
	"errors"
	"github.com/hankeyyh/a-simple-rpc/protocol"
	"github.com/hankeyyh/a-simple-rpc/share"
	"log"
	"net"
	"sync"
	"time"
)

const (
	// ReaderBuffSize is used for bufio reader.
	ReaderBuffSize = 16 * 1024
)

var (
	ErrShutdown         = errors.New("connection is shut down")
	ErrUnsupportedCodec = errors.New("unsupported codec")
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
		option:  option,
		pending: make(map[uint64]*Call),
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
		// byte->msg
		msg := protocol.NewMessage()
		err = msg.Decode(c.r)
		if err != nil {

		}

		// msg->call
		c.mutex.Lock()
		call := c.pending[msg.Seq()]
		c.mutex.Unlock()

		codec := share.Codecs[msg.SerializeType()]
		if codec == nil {
			call.Error = ErrUnsupportedCodec
			call.done()
			continue
		}
		err = codec.Decode(msg.Payload, call.Reply)
		if err != nil {
			call.Error = err
			call.done()
			continue
		}
		call.ResMetadata = msg.Metadata

		// call complete
		call.done()
	}
}

func (c *Client) heartbeat() {

}

func (c *Client) Go(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}, done chan *Call) *Call {
	// 生成call
	call := new(Call)
	call.ServicePath = servicePath
	call.ServiceMethod = serviceMethod
	call.Args = args
	call.Reply = reply

	meta := ctx.Value(share.ReqMetaDataKey)
	if meta != nil {
		call.Metadata = meta.(map[string]string)
	}

	if done == nil {
		done = make(chan *Call, 10)
	} else {
		if cap(done) == 0 {
			log.Panic("rpc: done channel is unbuffered")
		}
	}
	call.Done = done

	go c.send(ctx, call)

	return call
}

func (c *Client) send(ctx context.Context, call *Call) {
	c.mutex.Lock()
	// 客户端已经关闭
	if c.shutdown || c.closing {
		call.Error = ErrShutdown
		call.done()
		c.mutex.Unlock()
		return
	}

	// 序列化方式错误
	codec := share.Codecs[c.option.SerializeType]
	if codec == nil {
		call.Error = ErrUnsupportedCodec
		call.done()
		c.mutex.Unlock()
		return
	}

	// 缓存call
	seq := c.seq
	c.seq++
	c.pending[seq] = call

	c.mutex.Unlock()

	// call -> msg
	msg := protocol.NewMessage()
	msg.SetMessageType(protocol.Request)
	msg.SetSerializeType(c.option.SerializeType)
	msg.SetSeq(seq)
	msg.ServicePath = call.ServicePath
	msg.ServiceMethod = call.ServiceMethod
	msg.Metadata = call.Metadata

	payload, err := codec.Encode(call.Args)
	if err != nil {
		c.mutex.Lock()
		delete(c.pending, seq)
		c.mutex.Unlock()
		call.Error = err
		call.done()
		return
	}
	msg.Payload = payload

	msgRaw := msg.Encode()

	// 发送消息
	_, err = c.Conn.Write(msgRaw)
	if err != nil {
		c.mutex.Lock()
		delete(c.pending, seq)
		c.mutex.Unlock()
		call.Error = err
		call.done()
		return
	}

	if c.option.IdleTimeout != 0 {
		c.Conn.SetDeadline(time.Now().Add(c.option.IdleTimeout))
	}
}

func (c *Client) GetConn() net.Conn {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Call(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error {
	Done := c.Go(ctx, servicePath, serviceMethod, args, reply, make(chan *Call, 1)).Done

	var err error
	select {
	case <-ctx.Done():
		// 超时cancel
		c.mutex.Lock()
		// 删除call缓存
		c.mutex.Unlock()

	case call := <-Done:
		// 调用完成
		err = call.Error

	}
	return err
}

func (c *Client) Close() error {
	return c.Conn.Close()
}

func (c *Client) IsClosing() bool {
	//TODO implement me
	panic("implement me")
}

func (c *Client) IsShutdown() bool {
	//TODO implement me
	panic("implement me")
}
