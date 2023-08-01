package server

import (
	"context"
	"errors"
	"net"
	"reflect"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// Precompute the reflect type for error. Can't use error directly
// because Typeof takes an empty interface value. This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

// Precompute the reflect type for context.
var typeOfContext = reflect.TypeOf((*context.Context)(nil)).Elem()

type methodType struct {
	sync.Mutex
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name         string
	instance     reflect.Value // receiver of methods for the service
	instanceType reflect.Type  // type of the receiver
	methodMap    map[string]*methodType
}

type Server struct {
	// listener
	ln net.Listener

	//超时时间
	readTimeout  time.Duration
	writeTimeout time.Duration

	// service map， 锁
	serviceMap     map[string]*service
	serviceMapLock sync.RWMutex

	// 活跃连接缓存，锁
	actionConnMap     map[net.Conn]interface{}
	actionConnMapLock sync.RWMutex
}

type OptionFn func(s *Server)

func NewServer(options ...OptionFn) *Server {
	s := &Server{
		serviceMap:    make(map[string]*service),
		actionConnMap: make(map[net.Conn]interface{}),
	}

	for _, fn := range options {
		fn(s)
	}

	return s
}

func (svr *Server) Register(serviceInstance interface{}) error {
	svr.serviceMapLock.Lock()
	defer svr.serviceMapLock.Unlock()

	service := new(service)
	service.instanceType = reflect.TypeOf(serviceInstance)
	service.instance = reflect.ValueOf(serviceInstance)
	service.name = reflect.Indirect(service.instance).Type().Name()

	// 注册合法的方法
	validMethodMap := suitableRPCMethods(service.instanceType)
	if len(validMethodMap) == 0 {
		return errors.New("register: type " + service.name + " has no exported methods of suitable type")
	}
	service.methodMap = validMethodMap

	svr.serviceMap[service.name] = service
	return nil
}

func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.PkgPath() == "" || isExported(t.Name())
}

func suitableRPCMethods(typ reflect.Type) map[string]*methodType {
	methodMap := make(map[string]*methodType)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		// 方法必须被导出
		if method.PkgPath != "" {
			continue
		}
		mtype := method.Type

		if mtype.NumIn() != 4 {
			continue
		}

		// 第一个参数必须是context.Context
		ctxType := mtype.In(1)
		if !ctxType.Implements(typeOfContext) {
			continue
		}

		// 第二个参数必须是导出类型
		argType := mtype.In(2)
		if !isExportedOrBuiltinType(argType) {
			continue
		}

		// 第三个参数必须指针且导出类型
		replyType := mtype.In(3)
		if replyType.Kind() != reflect.Pointer {
			continue
		}
		if !isExportedOrBuiltinType(replyType) {
			continue
		}

		// 返回值只能有一个且是error类型
		if mtype.NumOut() != 1 {
			continue
		}
		returnType := mtype.Out(0)
		if returnType != typeOfError {
			continue
		}

		// 注册
		methodMap[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
	}
	return methodMap
}

func (svr *Server) Serve(network, address string) error {
	ln, err := makeListenerMap[network](address)
	if err != nil {
		return err
	}

	return nil
}
