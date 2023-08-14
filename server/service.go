package server

import (
	"context"
	"log"
	"reflect"
	"sync"
)

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
