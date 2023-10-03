package log

import (
	"testing"
)

type A struct {
	name string
	age  int
}

func TestDefaultLogger(t *testing.T) {
	a := A{"yuhan", 12}
	Debug(123, 234, a)
}
