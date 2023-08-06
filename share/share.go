package share

import (
	"github.com/hankeyyh/a-simple-rpc/codec"
	"github.com/hankeyyh/a-simple-rpc/protocol"
)

// 编码格式
var Codecs = map[protocol.SerializeType]codec.Codec{
	protocol.JSON:        &codec.JSONCodec{},
	protocol.ProtoBuffer: &codec.PBCodec{},
}
