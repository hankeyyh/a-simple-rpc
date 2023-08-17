package client

import "sync"

type KVPair struct {
	Key   string
	Value string
}

// 服务发现 etcd, consul, ...
type ServiceDiscovery interface {
	GetService() []*KVPair
	Close()
}

// 点对点服务发现
type Peer2PeerDiscovery struct {
	server   string
	metadata string
}

func NewPeer2PeerDiscovery(server, metadata string) (*Peer2PeerDiscovery, error) {
	return &Peer2PeerDiscovery{
		server:   server,
		metadata: metadata,
	}, nil
}

// 返回静态地址
func (p *Peer2PeerDiscovery) GetService() []*KVPair {
	return []*KVPair{
		{
			Key:   p.server,
			Value: p.metadata,
		},
	}
}

func (p *Peer2PeerDiscovery) Close() {

}

// 点对多寻址
type MultiServersDiscovery struct {
	pairsMu sync.RWMutex
	pairs   []*KVPair
}

func NewMultiServersDiscovery(pairs []*KVPair) *MultiServersDiscovery {
	return &MultiServersDiscovery{
		pairs: pairs,
	}
}

func (m *MultiServersDiscovery) GetService() []*KVPair {
	m.pairsMu.RLock()
	defer m.pairsMu.RUnlock()
	return m.pairs
}

func (m *MultiServersDiscovery) Close() {

}
