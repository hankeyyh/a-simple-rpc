package client

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/hankeyyh/a-simple-rpc/protocol"
	"github.com/hankeyyh/a-simple-rpc/share"
	"io"
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
	ErrShutdown                 = errors.New("connection is shut down")
	ErrUnsupportedCodec         = errors.New("unsupported codec")
	ErrUnsupportedMsgStatusType = errors.New("unsupported msg status type")
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
			break
		}

		// msg->call
		c.mutex.Lock()
		call := c.pending[msg.Seq()]
		delete(c.pending, msg.Seq())
		c.mutex.Unlock()

		// 可能call已经超时，已经被移除
		if call == nil {
			continue
		}

		// svr返回错误
		if msg.MessageStatusType() == protocol.Error {
			// 错误
			if len(msg.Metadata) > 0 {
				call.ResMetadata = msg.Metadata
				call.Error = NewServiceError(msg.Metadata[protocol.ServiceError])
			}
			// payload
			if len(msg.Payload) > 0 {
				codec := share.Codecs[msg.SerializeType()]
				if codec != nil {
					_ = codec.Decode(msg.Payload, call.Reply)
				}
			}
			call.done()
			continue
		} else if msg.MessageStatusType() == protocol.Normal {
			// metadata
			if len(msg.Metadata) > 0 {
				call.ResMetadata = msg.Metadata
			}
			// payload
			if len(msg.Payload) > 0 {
				codec := share.Codecs[msg.SerializeType()]
				if codec == nil {
					call.Error = NewServiceError(ErrUnsupportedCodec.Error())
				} else {
					err = codec.Decode(msg.Payload, call.Reply)
					if err != nil {
						call.Error = NewServiceError(err.Error())
					}
				}
			}
		} else {
			// msg状态无法识别
			call.Error = NewServiceError(ErrUnsupportedMsgStatusType.Error())
		}

		// call complete
		call.done()
	}

	// 出现异常，关闭所有pending call
	c.mutex.Lock()
	c.Conn.Close()
	c.shutdown = true
	if e, ok := err.(*net.OpError); ok {
		if e.Addr != nil || e.Err != nil {
			err = fmt.Errorf("net.Operror: %s", e.Err.Error())
		} else {
			err = errors.New("net.OpError")
		}
	}
	if err == io.EOF {
		if c.closing {
			err = ErrShutdown
		} else {
			err = io.ErrUnexpectedEOF
		}
	}
	for _, call := range c.pending {
		call.Error = err
		call.done()
	}
	c.mutex.Unlock()

	// 未知错误
	if err != nil && !c.closing {
		log.Printf("client protocol error: %v", err)
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

	// 缓存seq
	if cseq, ok := ctx.Value(share.SeqKey{}).(*uint64); ok {
		*cseq = seq
	}

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
	return c.Conn
}

func (c *Client) Call(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error {
	// ctx 缓存seq，当ctx被cancel时，需要根据seq删除pending call
	seq := new(uint64)
	ctx = context.WithValue(ctx, share.SeqKey{}, seq)

	Done := c.Go(ctx, servicePath, serviceMethod, args, reply, make(chan *Call, 1)).Done

	var err error
	select {
	case <-ctx.Done():
		c.mutex.Lock()
		call := c.pending[*seq]
		delete(c.pending, *seq)
		c.mutex.Unlock()
		if call != nil {
			call.Error = ctx.Err()
			call.done()
		}
		return ctx.Err()
	case call := <-Done:
		// 调用完成
		err = call.Error

	}
	return err
}

func (c *Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 缓存的call强制关闭
	for i, call := range c.pending {
		delete(c.pending, i)
		if call != nil {
			call.Error = ErrShutdown
			call.done()
		}
	}
	if c.closing || c.shutdown {
		return ErrShutdown
	}

	c.closing = true
	return c.Conn.Close()
}

func (c *Client) IsClosing() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.closing
}

func (c *Client) IsShutdown() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.shutdown
}
