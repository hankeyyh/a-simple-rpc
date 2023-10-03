package client

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
