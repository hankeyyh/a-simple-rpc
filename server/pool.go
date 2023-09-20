package server

import (
	"reflect"
	"sync"
)

type Reset interface {
	Reset()
}

var reflectTypePools = &typePools{
	pools: make(map[reflect.Type]*sync.Pool),
	New: func(t reflect.Type) interface{} {
		var argv reflect.Value
		if t.Kind() == reflect.Pointer {
			argv = reflect.New(t.Elem())
		} else {
			argv = reflect.New(t)
		}
		return argv.Interface()
	},
}

type typePools struct {
	mu    sync.RWMutex
	pools map[reflect.Type]*sync.Pool
	New   func(t reflect.Type) interface{}
}

func (p *typePools) Init(t reflect.Type) {
	tp := sync.Pool{
		New: func() interface{} {
			return p.New(t)
		},
	}
	p.mu.Lock()
	p.pools[t] = &tp
	p.mu.Unlock()
}

func (p *typePools) Put(t reflect.Type, x interface{}) {
	if o, ok := x.(Reset); ok {
		o.Reset()
		p.mu.RLock()
		pool := p.pools[t]
		p.mu.RUnlock()
		pool.Put(x)
	}
}

func (p *typePools) Get(t reflect.Type) interface{} {
	p.mu.RLock()
	pool := p.pools[t]
	p.mu.RUnlock()

	return pool.Get()
}

func (p *typePools) GetRaw(t reflect.Type) interface{} {
	var argv reflect.Value
	if t.Kind() == reflect.Pointer {
		argv = reflect.New(t.Elem())
	} else {
		argv = reflect.New(t)
	}
	return argv.Interface()
}
