package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	rpc_error "github.com/hankeyyh/a-simple-rpc/error"
	"github.com/hankeyyh/a-simple-rpc/protocol"
	"github.com/hankeyyh/a-simple-rpc/share"
	"github.com/soheilhy/cmux"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"
)

type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "rpcx context value " + k.name }

var (
	// ErrServerClosed is returned by the Server's Serve, ListenAndServe after a call to Shutdown or Close.
	ErrServerClosed  = errors.New("http: Server closed")
	ErrReqReachLimit = errors.New("request reached rate limit")

	typeOfError   = reflect.TypeOf((*error)(nil)).Elem()
	typeOfContext = reflect.TypeOf((*context.Context)(nil)).Elem()

	// 默认长连接时长
	DefaultKeepAliveDuration = 3 * time.Minute

	// 缓存连接
	RemoteConnContexKey = &contextKey{"remote-conn"}
	// 开始处理请求的时间
	StartRequestContextKey = &contextKey{"start-parse-request"}
	// 开始发送请求的时间
	StartSendRequestContextKey = &contextKey{"start-send-request"}
)

const (
	// ReaderBuffsize is used for bufio reader.
	ReaderBuffsize = 1024
)

type Server struct {
	// listener
	ln net.Listener

	//超时时间
	readTimeout  time.Duration
	writeTimeout time.Duration

	// service map， 锁
	serviceMap         map[string]*service
	serviceMapLock     sync.RWMutex
	serviceBasePathMap map[string]string // bathPath->serviceName

	// 插件
	Plugins PluginContainer

	// 活跃连接缓存，锁
	mu            sync.RWMutex
	actionConnMap map[net.Conn]interface{}
	doneChan      chan struct{}
	seq           atomic.Uint64

	inShutdown int32

	// 正在处理的请求数量
	handlerMsgNum int32

	// http server
	DisableHTTPGateway bool // 禁用http请求/响应
	gatewayHTTPServer  *http.Server
}

type OptionFn func(s *Server)

func NewServer(options ...OptionFn) *Server {
	s := &Server{
		Plugins:       &pluginContainer{},
		serviceMap:    make(map[string]*service),
		actionConnMap: make(map[net.Conn]interface{}),
		doneChan:      make(chan struct{}),
	}

	for _, fn := range options {
		fn(s)
	}

	return s
}

func (svr *Server) Register(serviceInstance interface{}) error {
	sname, err := svr.register("", serviceInstance)
	if err != nil {
		return err
	}
	return svr.Plugins.DoRegister(sname, serviceInstance)
}

func (svr *Server) RegisterName(serviceName string, serviceInstance interface{}) error {
	_, err := svr.register(serviceName, serviceInstance)
	if err != nil {
		return err
	}
	return svr.Plugins.DoRegister(serviceName, serviceInstance)
}

func (svr *Server) RegisterByProto(serviceName string, serviceInstance interface{}, sd *descriptor.ServiceDescriptorProto) error {
	_, err := svr.registerByProto(serviceName, serviceInstance, sd)
	if err != nil {
		return err
	}

	return svr.Plugins.DoRegister(serviceName, serviceInstance)
}

func (svr *Server) registerByProto(serviceName string, serviceInstance interface{}, sd *descriptor.ServiceDescriptorProto) (string, error) {
	svr.serviceMapLock.Lock()
	defer svr.serviceMapLock.Unlock()
	instanceType := reflect.TypeOf(serviceInstance)
	instanceValue := reflect.ValueOf(serviceInstance)

	sname := reflect.Indirect(instanceValue).Type().Name()
	if serviceName != "" {
		sname = serviceName
	}
	// 合法的方法
	validMethodMap := suitableRPCMethods(instanceType)
	if len(validMethodMap) == 0 {
		return sname, errors.New("register: type " + sname + " has no exported methods of suitable type")
	}

	// service 后处理，从sd解析http path
	option := withServiceHTTPRouteOption(sd)

	// 创建service
	svc := newService(sname, instanceType, instanceValue, validMethodMap, option)

	svr.serviceMap[svc.name] = svc
	svr.serviceBasePathMap[svc.bathPath] = svc.name

	return svc.name, nil
}

func (svr *Server) register(serviceName string, serviceInstance interface{}) (string, error) {
	svr.serviceMapLock.Lock()
	defer svr.serviceMapLock.Unlock()

	service := new(service)
	service.instanceType = reflect.TypeOf(serviceInstance)
	service.instance = reflect.ValueOf(serviceInstance)
	sname := reflect.Indirect(service.instance).Type().Name()
	if serviceName != "" {
		sname = serviceName
	}
	service.name = sname

	// 注册合法的方法
	validMethodMap := suitableRPCMethods(service.instanceType)
	if len(validMethodMap) == 0 {
		return sname, errors.New("register: type " + service.name + " has no exported methods of suitable type")
	}
	service.methodMap = validMethodMap

	svr.serviceMap[service.name] = service

	return sname, nil
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
		if !checkValidMethod(method) {
			continue
		}

		mtype := method.Type
		argType := mtype.In(2)
		replyType := mtype.In(3)

		// 注册
		methodMap[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}

		// 类型注册到pool中
		reflectTypePools.Init(argType)
		reflectTypePools.Init(replyType)
	}
	return methodMap
}

// 检查方法是否满足接口注册条件
func checkValidMethod(method reflect.Method) bool {
	// 方法必须被导出
	if method.PkgPath != "" {
		return false
	}
	mtype := method.Type

	if mtype.NumIn() != 4 {
		return false
	}

	// 第一个参数必须是context.Context
	ctxType := mtype.In(1)
	if !ctxType.Implements(typeOfContext) {
		return false
	}

	// 第二个参数必须是导出类型
	argType := mtype.In(2)
	if !isExportedOrBuiltinType(argType) {
		return false
	}

	// 第三个参数必须指针且导出类型
	replyType := mtype.In(3)
	if replyType.Kind() != reflect.Pointer {
		return false
	}
	if !isExportedOrBuiltinType(replyType) {
		return false
	}

	// 返回值只能有一个且是error类型
	if mtype.NumOut() != 1 {
		return false
	}
	returnType := mtype.Out(0)
	if returnType != typeOfError {
		return false
	}
	return true
}

func (svr *Server) Serve(network, address string) error {
	ln, err := makeListenerMap[network](address)
	if err != nil {
		return err
	}

	defer svr.UnregisterAll()

	// 启动gateway，支持http请求响应模型
	ln = svr.startGateway(network, ln)

	return svr.serveListener(ln)
}

// 创建一个service的goroutine来处理请求
func (svr *Server) serveListener(ln net.Listener) error {
	svr.mu.Lock()
	svr.ln = ln
	svr.mu.Unlock()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// 若server已关闭
			if svr.isShutdown() {
				<-svr.doneChan
				return ErrServerClosed
			}

			// 连接关闭
			if errors.Is(err, cmux.ErrListenerClosed) {
				return ErrServerClosed
			}

			return err
		}

		if tc, ok := conn.(*net.TCPConn); ok {
			// 设置tcp长连接
			tc.SetKeepAlive(true)
			tc.SetKeepAlivePeriod(DefaultKeepAliveDuration)
			tc.SetLinger(10)
		}

		svr.mu.Lock()
		svr.actionConnMap[conn] = struct{}{}
		svr.mu.Unlock()

		go svr.serveConn(conn)
	}
}

func (svr *Server) serveConn(conn net.Conn) {
	if svr.isShutdown() {
		svr.closeConn(conn)
		return
	}

	defer func() {
		if err := recover(); err != nil {
			log.Printf("serving %s panic error: %s", conn.RemoteAddr().String(), err)
		}
		// 关闭连接
		if svr.isShutdown() {
			<-svr.doneChan
		}
		svr.closeConn(conn)
	}()

	// 设置conn的读写超时
	if tlsConn, ok := conn.(*net.TCPConn); ok {
		if d := svr.readTimeout; d != 0 {
			tlsConn.SetReadDeadline(time.Now().Add(d))
		}
		if d := svr.writeTimeout; d != 0 {
			tlsConn.SetWriteDeadline(time.Now().Add(d))
		}
	}

	r := bufio.NewReaderSize(conn, ReaderBuffsize)
	// read requests and handle it
	for {
		if svr.isShutdown() {
			return
		}
		// 设置读超时
		if svr.readTimeout != 0 {
			conn.SetReadDeadline(time.Now().Add(svr.readTimeout))
		}

		// ctx缓存连接
		ctx := context.WithValue(context.Background(), RemoteConnContexKey, conn)

		req, err := svr.readRequest(ctx, r)
		if err != nil {
			if err == io.EOF {
				log.Printf("client has closed this connection: %s", conn.RemoteAddr().String())
			} else if errors.Is(err, net.ErrClosed) {
				log.Printf("connection %s is closed", conn.RemoteAddr().String())
			} else if errors.Is(err, ErrReqReachLimit) {
				if !req.IsOneway() {
					// 请求达到上限
					res := req.Clone()
					res.SetMessageType(protocol.Response)

					svr.handleError(res, err)
					svr.sendResponse(conn, res)
				}
				continue
			} else {
				log.Printf("failed to read request: %v", err)
			}
			return
		}

		ctx = context.WithValue(ctx, StartRequestContextKey, time.Now().UnixNano())

		// 请求检验
		if !req.IsHeartbeat() {
			err = svr.auth(ctx, req)
			if err != nil {
				log.Printf("auth failed for conn %s: %v", conn.RemoteAddr().String(), err)
				if !req.IsOneway() {
					res := req.Clone()
					res.SetMessageType(protocol.Response)
					svr.handleError(res, err)
					svr.sendResponse(conn, res)
				}
				return
			}
		}

		// 处理请求
		go svr.processOneRequest(ctx, req, conn)
	}
}

// 处理请求
func (svr *Server) processOneRequest(ctx context.Context, req *protocol.Message, conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("failed to process the request: %v", r)
		}
	}()

	// 记录当前处理的请求数量
	atomic.AddInt32(&svr.handlerMsgNum, 1)
	defer atomic.AddInt32(&svr.handlerMsgNum, -1)

	// 心跳请求，直接返回
	if req.IsHeartbeat() {
		log.Printf("recv heartbeat")
		req.SetMessageType(protocol.Response)
		data := req.Encode()
		if svr.writeTimeout != 0 {
			conn.SetWriteDeadline(time.Now().Add(svr.writeTimeout))
		}
		conn.Write(data)
		return
	}

	res, err := svr.handleRequest(ctx, req)
	if err != nil {
		log.Printf("failed to handle request: %v", err)
	}

	if !req.IsOneway() {
		svr.sendResponse(conn, res)
	}
}

func (svr *Server) getService(servicePath string) (*service, error) {
	svr.serviceMapLock.RLock()
	svc := svr.serviceMap[servicePath]
	svr.serviceMapLock.RUnlock()
	if svc == nil {
		return nil, errors.New("can't find service " + servicePath)
	}
	return svc, nil
}

func (svr *Server) getServiceByBathPath(serviceBathPath string) (*service, error) {
	svr.serviceMapLock.RLock()
	svcName := svr.serviceBasePathMap[serviceBathPath]
	svc := svr.serviceMap[svcName]
	svr.serviceMapLock.RUnlock()
	if svc == nil {
		return nil, errors.New("can't find service " + serviceBathPath)
	}
	return svc, nil
}

// 处理请求
func (svr *Server) handleRequest(ctx context.Context, req *protocol.Message) (res *protocol.Message, err error) {
	// 请求服务，方法
	servicePath := req.ServicePath
	serviceMethod := req.ServiceMethod
	res = req.Clone()
	res.SetMessageType(protocol.Response)

	// 对应的服务
	svc, err := svr.getService(servicePath)
	if err != nil {
		return svr.handleError(res, err)
	}

	// 对应的方法
	methodTyp := svc.methodMap[serviceMethod]
	if methodTyp == nil {
		err = errors.New("can't find method " + serviceMethod)
		return svr.handleError(res, err)
	}

	// 获取方法的请求参数实例
	argv := reflectTypePools.Get(methodTyp.ArgType)

	// 反序列化
	codec := share.Codecs[req.SerializeType()]
	if codec == nil {
		err = fmt.Errorf("can not find codec for %d", req.SerializeType())
		return svr.handleError(res, err)
	}
	err = codec.Decode(req.Payload, argv)
	if err != nil {
		return svr.handleError(res, err)
	}

	// reply
	replyv := reflectTypePools.Get(methodTyp.ReplyType)

	// call service method
	if methodTyp.ArgType.Kind() != reflect.Pointer {
		err = svc.call(ctx, methodTyp, reflect.ValueOf(argv).Elem(), reflect.ValueOf(replyv))
	} else {
		err = svc.call(ctx, methodTyp, reflect.ValueOf(argv), reflect.ValueOf(replyv))
	}

	// 重新将argv, replyv 加入缓存
	reflectTypePools.Put(methodTyp.ArgType, argv)

	if err != nil {
		if replyv != nil {
			var payload []byte
			payload, err = codec.Encode(replyv)
			reflectTypePools.Put(methodTyp.ReplyType, replyv)
			if err != nil {
				return svr.handleError(res, err)
			}
			res.Payload = payload
		}
		return svr.handleError(res, err)
	}

	// 返回值reply->msg
	if !req.IsOneway() {
		var payload []byte
		payload, err = codec.Encode(replyv)
		reflectTypePools.Put(methodTyp.ReplyType, replyv)
		if err != nil {
			return svr.handleError(res, err)
		}
		res.Payload = payload
	} else if replyv != nil {
		reflectTypePools.Put(methodTyp.ReplyType, replyv)
	}

	return res, nil
}

// 读取请求
func (svr *Server) readRequest(ctx context.Context, r io.Reader) (req *protocol.Message, err error) {
	req, err = protocol.Read(r)
	return
}

func (svr *Server) isShutdown() bool {
	return atomic.LoadInt32(&svr.inShutdown) == 1
}

func (svr *Server) closeConn(conn net.Conn) error {
	svr.mu.Lock()
	delete(svr.actionConnMap, conn)
	svr.mu.Unlock()

	return conn.Close()
}

// 立即关闭
func (svr *Server) Close() error {
	svr.mu.Lock()
	defer svr.mu.Unlock()

	var err error
	// 关闭listener
	if svr.ln != nil {
		err = svr.ln.Close()
	}
	// 关闭conn
	for c := range svr.actionConnMap {
		c.Close()
		delete(svr.actionConnMap, c)
	}
	// 关闭http gateway
	if svr.gatewayHTTPServer != nil {
		svr.gatewayHTTPServer.Close()
	}
	// 关闭done chan
	svr.closeDoneChanLocked()

	return err
}

func (svr *Server) closeDoneChanLocked() {
	select {
	case <-svr.doneChan:
		// already closed
	default:
		close(svr.doneChan)
	}
}

// rsp中添加错误信息
func (svr *Server) handleError(res *protocol.Message, err error) (*protocol.Message, error) {
	res.SetMessageStatusType(protocol.Error)
	res.Metadata[protocol.ServiceError] = err.Error()
	return res, err
}

func (svr *Server) sendResponse(conn net.Conn, res *protocol.Message) {
	data := res.Encode()
	if svr.writeTimeout != 0 {
		conn.SetWriteDeadline(time.Now().Add(svr.writeTimeout))
	}
	conn.Write(data)
}

// 校验请求
func (svr *Server) auth(ctx context.Context, req *protocol.Message) error {
	// todo 校验函数
	return nil
}

// unregisters all services.
func (svr *Server) UnregisterAll() error {
	svr.serviceMapLock.RLock()
	defer svr.serviceMapLock.RUnlock()
	var errs []error
	for sname := range svr.serviceMap {
		if err := svr.Plugins.DoUnRegister(sname); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return rpc_error.NewMultiError(errs)
	}

	return nil
}
