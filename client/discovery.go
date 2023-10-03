package client

type KVPair struct {
	Key   string
	Value string
}

// 服务发现 etcd, consul, ...
type ServiceDiscovery interface {
	GetService() []*KVPair
	WatchService() chan []*KVPair
	RemoveWatcher(ch chan []*KVPair)
	Close()
}
