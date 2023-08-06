package codec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	pb "google.golang.org/protobuf/proto"
)

type Codec interface {
	Encode(i interface{}) ([]byte, error)
	Decode(data []byte, i interface{}) error
}

// protobuff
type PBCodec struct{}

func (P PBCodec) Encode(i interface{}) ([]byte, error) {
	if m, ok := i.(proto.Marshaler); ok {
		return m.Marshal()
	}
	if m, ok := i.(pb.Message); ok {
		return pb.Marshal(m)
	}
	return nil, fmt.Errorf("%T is not a proto.Marshaler or pb.Message", i)
}

func (P PBCodec) Decode(data []byte, i interface{}) error {
	if m, ok := i.(proto.Unmarshaler); ok {
		return m.Unmarshal(data)
	}
	if m, ok := i.(pb.Message); ok {
		return pb.Unmarshal(data, m)
	}
	return fmt.Errorf("%T is not a proto.Unmarshaler  or pb.Message", i)
}

// json
type JSONCodec struct{}

func (J JSONCodec) Encode(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}

func (J JSONCodec) Decode(data []byte, i interface{}) error {
	d := json.NewDecoder(bytes.NewBuffer(data))
	d.UseNumber()
	return d.Decode(i)
}
