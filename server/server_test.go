package server

import (
	"context"
	"fmt"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	"reflect"
	"testing"
)

type ArithImp struct {
}

func (s *ArithImp) Add(ctx context.Context, args *proto.Args, reply *proto.Reply) error {
	c := args.GetA() + args.GetB()
	reply.C = &c
	return nil
}

func (s *ArithImp) Mul(ctx context.Context, args *proto.Args, reply *proto.Reply) error {
	c := args.GetA() * args.GetB()
	reply.C = &c
	return nil
}

type MyError struct {
}

func (me MyError) Error() string {
	return ""
}

func TestCheckMethod(t *testing.T) {
	arith := new(ArithImp)
	instanceType := reflect.TypeOf(arith)
	fmt.Println("instanceType: ", instanceType)
	instanceTypeName := instanceType.Name()
	fmt.Println("instanceTypeName: ", instanceTypeName)
	instanceValue := reflect.ValueOf(arith)
	fmt.Println("instanceValue: ", instanceValue)
	serviceName := reflect.Indirect(instanceValue).Type().Name()
	fmt.Println("serviceName: ", serviceName)
	pkgPath := instanceType.PkgPath()
	fmt.Println("pkgPath: ", pkgPath)

	me := new(MyError)
	fmt.Println(reflect.TypeOf(me))

	for i := 0; i < instanceType.NumMethod(); i++ {
		method := instanceType.Method(i)
		mtype := method.Type
		fmt.Println("method.Type: ", mtype)

		ctxType := mtype.In(1)
		fmt.Println("ctxType valid: ", ctxType.Implements(typeOfContext))
		//argType := mtype.In(2)
		//
		//replyType := mtype.In(3)
	}
}
