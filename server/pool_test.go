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
	args := &Args{}
	reflectTypePools.Put(reflect.TypeOf(args), args)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reflectTypePools.Get(reflect.TypeOf(args))
	}
	b.StopTimer()
}

func BenchmarkTypePools_GetRaw(b *testing.B) {
	args := &Args{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reflectTypePools.GetRaw(reflect.TypeOf(args))
	}
	b.StopTimer()
}
