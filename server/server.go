package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/hankeyyh/a-simple-rpc/protocol"
	"github.com/hankeyyh/a-simple-rpc/share"
	"github.com/soheilhy/cmux"
	"io"
	"log"
	"net"
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
	serviceMap     map[string]*service
	serviceMapLock sync.RWMutex

	// 活跃连接缓存，锁
	mu            sync.RWMutex
	actionConnMap map[net.Conn]interface{}
	doneChan      chan struct{}
	seq           atomic.Uint64

	inShutdown int32

	// 正在处理的请求数量
	handlerMsgNum int32
}

type OptionFn func(s *Server)

func NewServer(options ...OptionFn) *Server {
	s := &Server{
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

		// 类型注册到pool中
		reflectTypePools.Init(argType)
		reflectTypePools.Init(replyType)
	}
	return methodMap
}

func (svr *Server) Serve(network, address string) error {
	ln, err := makeListenerMap[network](address)
	if err != nil {
		return err
	}

	defer svr.UnregisterAll()

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
		res := req.Clone()
		res.SetMessageType(protocol.Response)
		res.Payload = protocol.StringToSliceByte("OK")
		svr.sendResponse(conn, res)
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

// 处理请求
func (svr *Server) handleRequest(ctx context.Context, req *protocol.Message) (res *protocol.Message, err error) {
	// 请求服务，方法
	servicePath := req.ServicePath
	serviceMethod := req.ServiceMethod
	res = req.Clone()

	// 对应的服务
	svr.serviceMapLock.RLock()
	svc := svr.serviceMap[servicePath]
	svr.serviceMapLock.RUnlock()
	if svc == nil {
		err = errors.New("can't find service " + servicePath)
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
	if replyv != nil {
		reflectTypePools.Put(methodTyp.ReplyType, replyv)
	}

	if err != nil {
		if replyv != nil {
			var payload []byte
			if payload, err = codec.Encode(replyv); err != nil {
				return svr.handleError(res, err)
			}
			res.Payload = payload
		}
		return svr.handleError(res, err)
	}

	// 返回值reply->msg
	if !req.IsOneway() {
		var payload []byte
		if payload, err = codec.Encode(replyv); err != nil {
			return svr.handleError(res, err)
		}
		res.Payload = payload
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
	// todo plugin unregister service
	return nil
}
