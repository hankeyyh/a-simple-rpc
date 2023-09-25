package server

import (
	"context"
	rpc_error "github.com/hankeyyh/a-simple-rpc/error"
	"net"
)

// plugin 容器接口
type PluginContainer interface {
	Add(plugin Plugin)
	Remove(plugin Plugin)
	All() []Plugin

	DoRegister(name string, rcvr interface{}, metadata string) error
	DoUnRegister(name string) error

	DoPostConnAccept(conn net.Conn) (net.Conn, bool)
	DoPostConnClose(conn net.Conn) bool

	DoPreCall(ctx context.Context, serviceName, methodName string, args interface{}) (interface{}, error)
}

type (
	RegisterPlugin interface {
		Register(name string, rcvr interface{}, metadata string) error
		UnRegister(name string) error
	}

	PreCallPlugin interface {
		PreCall(ctx context.Context, serviceName, methodName string, args interface{}) (interface{}, error)
	}

	// 在连接accept后处理，当返回false时，连接关闭，后续的plugin不再继续处理
	PostConnAcceptPlugin interface {
		HandleConnAccept(conn net.Conn) (net.Conn, bool)
	}

	// 在连接close后处理
	PostConnClosePlugin interface {
		HandleConnClose(conn net.Conn) bool
	}
)

type Plugin interface{}

type pluginContainer struct {
	plugins []Plugin
}

func (p *pluginContainer) Add(plugin Plugin) {
	p.plugins = append(p.plugins, plugin)
}

func (p *pluginContainer) Remove(plugin Plugin) {
	if p == nil {
		return
	}
	plugins := make([]Plugin, 0, len(p.plugins))
	for _, pp := range p.plugins {
		if pp != plugin {
			plugins = append(plugins, pp)
		}
	}
	p.plugins = plugins
}

func (p *pluginContainer) All() []Plugin {
	return p.plugins
}

func (p *pluginContainer) DoRegister(name string, rcvr interface{}, metadata string) error {
	var errs []error
	for _, pp := range p.plugins {
		if rp, ok := pp.(RegisterPlugin); ok {
			err := rp.Register(name, rcvr, metadata)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return rpc_error.NewMultiError(errs)
	}
	return nil
}

func (p *pluginContainer) DoUnRegister(name string) error {
	var errs []error
	for _, pp := range p.plugins {
		if rp, ok := pp.(RegisterPlugin); ok {
			err := rp.UnRegister(name)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return rpc_error.NewMultiError(errs)
	}
	return nil
}

func (p *pluginContainer) DoPostConnAccept(conn net.Conn) (net.Conn, bool) {
	var flag bool
	for _, pp := range p.plugins {
		if ap, ok := pp.(PostConnAcceptPlugin); ok {
			conn, flag = ap.HandleConnAccept(conn)
			if !flag {
				conn.Close()
				return conn, false
			}
		}
	}

	return conn, true
}

func (p *pluginContainer) DoPostConnClose(conn net.Conn) bool {
	var flag bool
	for _, pp := range p.plugins {
		if cp, ok := pp.(PostConnClosePlugin); ok {
			flag = cp.HandleConnClose(conn)
			if !flag {
				return false
			}
		}
	}
	return true
}

func (p *pluginContainer) DoPreCall(ctx context.Context, serviceName, methodName string, args interface{}) (interface{}, error) {
	var err error
	for _, pp := range p.plugins {
		if plugin, ok := pp.(PreCallPlugin); ok {
			args, err = plugin.PreCall(ctx, serviceName, methodName, args)
			if err != nil {
				return args, err
			}
		}
	}
	return args, nil
}
