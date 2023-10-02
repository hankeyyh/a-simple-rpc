package client

import "sync"

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

// 点对点服务发现
type Peer2PeerDiscovery struct {
	server   string
	metadata string
}

func (p *Peer2PeerDiscovery) WatchService() chan []*KVPair {
	return nil
}

func (p *Peer2PeerDiscovery) RemoveWatcher(ch chan []*KVPair) {

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
	chans   []chan []*KVPair

	mu sync.Mutex
}

func (m *MultiServersDiscovery) WatchService() chan []*KVPair {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan []*KVPair, 10)
	m.chans = append(m.chans, ch)
	return ch
}

func (m *MultiServersDiscovery) RemoveWatcher(ch chan []*KVPair) {
	m.mu.Lock()
	defer m.mu.Unlock()

	chans := make([]chan []*KVPair, 0)
	for _, cc := range m.chans {
		if cc == ch {
			continue
		}
		chans = append(chans, cc)
	}
	m.chans = chans
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
