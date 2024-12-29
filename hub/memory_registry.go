package hub

import (
	"fmt"
	"sync"
)

// InMemoryRegistry provides a simple in-memory storage implementation
type InMemoryRegistry struct {
	agents map[string]*AgentFile
	mutex  sync.RWMutex
}

func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		agents: make(map[string]*AgentFile),
	}
}

func (r *InMemoryRegistry) Store(name, version string, agent *AgentFile) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", name, version)
	r.agents[key] = agent
	return nil
}

func (r *InMemoryRegistry) Get(name, version string) (*AgentFile, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	key := fmt.Sprintf("%s:%s", name, version)
	agent, exists := r.agents[key]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", key)
	}
	return agent, nil
}

func (r *InMemoryRegistry) Close() error {
	return nil
}
