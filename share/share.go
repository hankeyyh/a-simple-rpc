package share

import (
	"github.com/hankeyyh/a-simple-rpc/codec"
	"github.com/hankeyyh/a-simple-rpc/protocol"
)

// ctx缓存seq
type SeqKey struct{}

// ContextKey defines key type in context.
type ContextKey string

// ReqMetaDataKey is used to set metadata in context of requests.
var ReqMetaDataKey = ContextKey("__req_metadata")

// ResMetaDataKey is used to set metadata in context of responses.
var ResMetaDataKey = ContextKey("__res_metadata")

// 编码格式
var Codecs = map[protocol.SerializeType]codec.Codec{
	protocol.JSON:        &codec.JSONCodec{},
	protocol.ProtoBuffer: &codec.PBCodec{},
}
