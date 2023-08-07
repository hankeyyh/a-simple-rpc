package client

import (
	"github.com/hankeyyh/a-simple-rpc/protocol"
	"time"
)

// 客户端配置
type Option struct {
	Retries             int
	ConnectTimeout      time.Duration
	SerializeType       protocol.SerializeType
	MaxWaitForHeartbeat time.Duration
	TCPKeepAlivePeriod  time.Duration

	// 服务实例是坏的，等待一段时间后重新尝试选择
	TimeToDisallow time.Duration
}

var DefaultOption = Option{
	Retries:             3,
	ConnectTimeout:      time.Second,
	SerializeType:       protocol.ProtoBuffer,
	MaxWaitForHeartbeat: 30 * time.Second,
	TCPKeepAlivePeriod:  time.Minute,
	TimeToDisallow:      time.Minute,
}
