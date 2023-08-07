package client

import (
	"context"
	"math/rand"
	"sync"
)

type Selector interface {
	Select(ctx context.Context, servicePath, serviceMethod string, args interface{}) string
	UpdateServer(servers map[string]string)
}

func newSelector(mode SelectMode, servers map[string]string) Selector {
	switch mode {
	case RandomSelect:
		return newRandomSelector(servers)
	default:
		return newRandomSelector(servers)
	}
}

// randomSelector selects randomly.
type randomSelector struct {
	mu      sync.RWMutex
	servers []string
}

func newRandomSelector(servers map[string]string) Selector {
	ss := make([]string, 0, len(servers))
	for k := range servers {
		ss = append(ss, k)
	}

	return &randomSelector{servers: ss}
}

func (s *randomSelector) Select(ctx context.Context, servicePath, serviceMethod string, args interface{}) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ss := s.servers
	if len(ss) == 0 {
		return ""
	}
	i := rand.Int31n(int32(len(s.servers)))
	return ss[i]
}

func (s *randomSelector) UpdateServer(servers map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ss := make([]string, 0, len(servers))
	for k := range servers {
		ss = append(ss, k)
	}

	s.servers = ss
}
