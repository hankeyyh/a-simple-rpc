package test

import (
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/hankeyyh/a-simple-rpc/test/proto"
	pb "google.golang.org/protobuf/proto"
	"log"
	"testing"
)

func TestDescriptor(t *testing.T) {
	for i := 0; i < proto.File_arith_proto.Services().Len(); i++ {
		svc := proto.File_arith_proto.Services().Get(i)
		for j := 0; j < svc.Methods().Len(); j++ {
			method := svc.Methods().Get(j)
			op, ok := method.Options().(*descriptor.MethodOptions)
			if ok {
				v := pb.GetExtension(op, proto.E_MethodOptionHttpApi).(*proto.HttpRouteOptions)
				url := v.GetPost()
				log.Print(url)
			}
		}
	}
}
