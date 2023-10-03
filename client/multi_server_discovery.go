package client

import "sync"

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
