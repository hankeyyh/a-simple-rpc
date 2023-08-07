package client

// 当调用服务失败时，如何处理
type FailMode int

const (
	// 选择另一个服务
	FailOver FailMode = iota

	// 快速失败，直接返回
	FailFast

	// 使用当前服务实例重试
	FailTry

	// 使用备用服务
	FailBackup
)

// 服务选择策略
type SelectMode int

const (
	// 随机
	RandomSelect SelectMode = iota

	// 轮询
	RoundRobin

	// 加权轮询
	WeightedRoundRobin

	// 加权ping时间
	WeightedICMP

	// 一致性哈希
	ConsistentHash

	// 地理最近
	Closest

	// 自定义策略
	SelectByUser = 1000
)
