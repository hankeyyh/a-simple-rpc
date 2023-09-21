package server

import (
	"reflect"
	"testing"
)

type Args struct {
	A string
	B string
}

func (a *Args) Reset() {
	a.A = ""
	a.B = ""
}

func BenchmarkTypePools_Get(b *testing.B) {
	args := &Args{A: "yuhan", B: "yang"}
	argType := reflect.TypeOf(args)
	reflectTypePools.Init(argType)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reflectTypePools.Put(argType, args)
			reflectTypePools.Get(reflect.TypeOf(args))
		}
	})
	b.StopTimer()
}

func BenchmarkTypePools_GetRaw(b *testing.B) {
	args := &Args{A: "yuhan", B: "yang"}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reflectTypePools.GetRaw(reflect.TypeOf(args))
		}
	})
	b.StopTimer()
}
