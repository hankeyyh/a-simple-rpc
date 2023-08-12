package client

import (
	"context"
	"errors"
	"golang.org/x/sync/singleflight"
	"log"
	"strings"
	"sync"
)

var (
	// ErrXClientShutdown xclient is shutdown.
	ErrXClientShutdown = errors.New("xClient is shut down")
	// ErrXClientNoServer selector can't found one server.
	ErrXClientNoServer = errors.New("can not found any server")
	// ErrServerUnavailable selected server is unavailable.
	ErrServerUnavailable = errors.New("selected server is unavailable")
)

// 来自服务端的错误
type ServiceError struct {
	errMsg string
}

func (e ServiceError) Error() string {
	return e.errMsg
}

func NewServiceError(msg string) ServiceError {
	return ServiceError{errMsg: msg}
}

type XClient interface {
	Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, done chan *Call) (*Call, error)
	Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error
	Close() error
}

type xClient struct {
	servicePath string
	option      Option

	// 服务发现
	failMode   FailMode
	selectMode SelectMode
	discovery  ServiceDiscovery
	selector   Selector

	mu      sync.RWMutex
	servers map[string]string

	// 缓存到服务实例的连接
	cachedClient map[string]RPCClient

	isShutdown bool

	slGroup singleflight.Group
}

func NewXClient(servicePath string, failMode FailMode, selectMode SelectMode, discovery ServiceDiscovery, option Option) XClient {
	client := &xClient{
		servicePath:  servicePath,
		option:       option,
		failMode:     failMode,
		selectMode:   selectMode,
		discovery:    discovery,
		servers:      make(map[string]string),
		cachedClient: make(map[string]RPCClient),
	}

	// 服务发现
	pairs := discovery.GetService()
	for _, p := range pairs {
		client.servers[p.Key] = p.Value
	}

	// 选择器
	client.selector = newSelector(selectMode, client.servers)

	return client
}

func (x *xClient) Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, done chan *Call) (*Call, error) {
	//TODO implement me
	panic("implement me")
}

func (x *xClient) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	if x.isShutdown {
		return ErrXClientShutdown
	}

	// todo 设置请求超时时间

	// 选择client
	k, client, err := x.selectClient(ctx, x.servicePath, serviceMethod, args)

	if err != nil && x.failMode == FailFast {
		return err
	}

	switch x.failMode {
	case FailTry: // 尝试再次访问相同实例
		retries := x.option.Retries
		for retries >= 0 {
			retries--

			if client != nil {
				err = x.wrapCall(ctx, client, serviceMethod, args, reply)
				if err == nil {
					return nil
				}
				// 调用超时或服务端返回err，直接返回
				if !uncoverError(err) {
					return err
				}
			}

			// 重新尝试获取client
			if uncoverError(err) {
				x.removeClient(k, client)
			}
			client, err = x.getCachedClient(k)
		}
		return err
	case FailOver: // 尝试下一个服务实例
		retries := x.option.Retries
		for retries >= 0 {
			retries--

			if client != nil {
				err = x.wrapCall(ctx, client, serviceMethod, args, reply)
				if err == nil {
					return nil
				}
				// 调用超时或服务端返回err，直接返回
				if !uncoverError(err) {
					return err
				}
			}

			// 重新尝试获取client
			if uncoverError(err) {
				x.removeClient(k, client)
			}
			k, client, err = x.selectClient(ctx, x.servicePath, serviceMethod, args)
		}
		return err
	case FailFast:
		err = x.wrapCall(ctx, client, serviceMethod, args, reply)
		if err != nil {
			if uncoverError(err) {
				x.removeClient(k, client)
			}
		}
		return err
	default:
		log.Printf("failMode %v not supported", x.failMode)
		return errors.New("failMode not supported")
	}
}

// 根据选择器算法，选择一个server
func (x *xClient) selectClient(ctx context.Context, servicePath, serviceMethod string, args interface{}) (string, RPCClient, error) {
	// 获得服务地址
	k := x.selector.Select(ctx, servicePath, serviceMethod, args)
	if k == "" {
		return "", nil, ErrXClientNoServer
	}

	client, err := x.getCachedClient(k)
	return k, client, err
}

// 获取缓存的client，如果没有则创建
func (x *xClient) getCachedClient(k string) (RPCClient, error) {
	if x.isShutdown {
		return nil, ErrXClientShutdown
	}

	// 检查缓存中的client
	x.mu.Lock()
	client := x.cachedClient[k]
	if client != nil {
		// 找到未关闭的client直接返回
		if !client.IsClosing() && !client.IsShutdown() {
			x.mu.Unlock()
			return client, nil
		}
		// client 已关闭状态，从缓存中移除
		x.removeClient(k, client)
		client = nil
	}
	x.mu.Unlock()

	// 创建新的client
	if client == nil {
		// 生成client，避免并发导致多次生成
		generatedClient, err, _ := x.slGroup.Do(k, func() (interface{}, error) {
			return x.generateClient(k)
		})

		if err != nil {
			x.slGroup.Forget(k)
			return nil, err
		}

		client = generatedClient.(RPCClient)

		// 加入缓存
		x.mu.Lock()
		x.cachedClient[k] = client
		x.mu.Unlock()

		x.slGroup.Forget(k)
	}
	return client, nil
}

// 生成client
func (x *xClient) generateClient(k string) (RPCClient, error) {
	client := NewClient(x.option)
	network, addr := splitNetworkAndAddress(k)

	err := client.Connect(network, addr)
	// todo 断路器fail

	return client, err
}

// 包装调用RPCClient.Call
func (x *xClient) wrapCall(ctx context.Context, client RPCClient, method string, args interface{}, reply interface{}) error {
	if client == nil {
		return ErrServerUnavailable
	}

	err := client.Call(ctx, x.servicePath, method, args, reply)
	return err
}

// 移除client
func (x *xClient) removeClient(k string, client RPCClient) {
	// 从缓存中移除
	x.mu.Lock()
	if client == x.cachedClient[k] {
		delete(x.cachedClient, k)
	}
	x.mu.Unlock()

	// 连接关闭
	if client != nil {
		client.Close()
	}
}

func uncoverError(err error) bool {
	var serviceError ServiceError
	if errors.As(err, &serviceError) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	if errors.Is(err, context.Canceled) {
		return false
	}

	return true
}

func splitNetworkAndAddress(server string) (string, string) {
	ss := strings.SplitN(server, "@", 2)
	if len(ss) == 1 {
		return "tcp", server
	}

	return ss[0], ss[1]
}

// 关闭管理的所有连接
func (x *xClient) Close() error {
	x.mu.Lock()
	x.isShutdown = true
	for k, client := range x.cachedClient {
		client.Close()
		delete(x.cachedClient, k)
	}
	x.mu.Unlock()

	return nil
}
