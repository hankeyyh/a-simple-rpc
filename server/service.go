package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	srpc "github.com/hankeyyh/a-simple-srpc"
	"google.golang.org/protobuf/proto"
	"log"
	"reflect"
	"strings"
	"sync"
)

type methodType struct {
	sync.Mutex
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
	// url path
	GetPath  string
	PostPath string
}

func (mt *methodType) SetPath(httpMethod string, path string) {
	if httpMethod == "GET" {
		mt.GetPath = path
	} else if httpMethod == "POST" {
		mt.PostPath = path
	}
}

type service struct {
	name         string
	bathPath     string        // url
	instance     reflect.Value // receiver of methods for the service
	instanceType reflect.Type  // type of the receiver
	methodMap    map[string]*methodType
	getPathMap   map[string]string // GET path -> methodName
	postPathMap  map[string]string // POST path -> methodName
}

type ServiceOptionFn func(s *service)

// 用于从proto文件注册服务时，添加 http path
func withServiceHTTPRouteOption(sd *descriptor.ServiceDescriptorProto) ServiceOptionFn {
	serviceBathPathOption := proto.GetExtension(sd.Options, srpc.E_ServiceBasePath)
	serviceBathPath := *(serviceBathPathOption.(*string))

	getPathMap := make(map[string]string)
	postPathMap := make(map[string]string)

	for _, mdp := range sd.Method {
		methodName := mdp.GetName()

		option := proto.GetExtension(mdp.Options, srpc.E_MethodOptionHttpApi)
		httpOption := option.(*srpc.HttpRouteOptions)

		if httpOption.Get != nil {
			// GET path
			path := httpOption.GetGet()
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			path = strings.TrimSuffix(path, "/")
			getPathMap[path] = methodName
		}
		if httpOption.Post != nil {
			// POST path
			path := httpOption.GetPost()
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			path = strings.TrimSuffix(path, "/")
			postPathMap[path] = methodName
		}
	}

	return func(s *service) {
		s.bathPath = serviceBathPath
		s.getPathMap = getPathMap
		s.postPathMap = postPathMap

		for getPath, methodName := range getPathMap {
			method := s.methodMap[methodName]
			method.SetPath("GET", getPath)
		}
		for postPath, methodName := range postPathMap {
			method := s.methodMap[methodName]
			method.SetPath("POST", postPath)
		}
	}
}

func newService(serviceName string, instanceType reflect.Type, instanceValue reflect.Value,
	methodMap map[string]*methodType, ops ...ServiceOptionFn) *service {
	svc := new(service)
	svc.name = serviceName
	svc.instanceType = instanceType
	svc.instance = instanceValue
	svc.methodMap = methodMap

	for _, op := range ops {
		op(svc)
	}
	return svc
}

func (s *service) call(ctx context.Context, mtype *methodType, argv, replyv reflect.Value) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[service internal error]: %v, method: %s, argv: %+v",
				r, mtype.method.Name, argv.Interface())
		}
	}()

	function := mtype.method.Func
	// invoke method, providing a new value for reply
	returnValues := function.Call([]reflect.Value{s.instance, reflect.ValueOf(ctx), argv, replyv})
	errInter := returnValues[0].Interface()
	if errInter != nil {
		return errInter.(error)
	}
	return nil
}

func (s *service) getMethod(name string) (*methodType, error) {
	mt, ok := s.methodMap[name]
	if !ok {
		return nil, errors.New("can't find method " + name)
	}
	return mt, nil
}

func (s *service) getMethodByHttpPath(httpMethod, httpPath string) (*methodType, error) {
	name := ""
	if httpMethod == "GET" {
		name = s.getPathMap[httpPath]
	}
	if httpMethod == "POST" {
		name = s.postPathMap[httpPath]
	}
	if name == "" {
		return nil, errors.New(fmt.Sprintf("can't find method, httpMethod: %s, httpPath: %s",
			httpMethod, httpPath))
	}
	return s.getMethod(name)
}
