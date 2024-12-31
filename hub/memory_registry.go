package hub

import (
	"encoding/json"
	"fmt"
	"sync"
)

type AgentVersions map[string]*AgentFile

type InMemoryRegistry struct {
	agents map[string]AgentVersions
	mutex  sync.RWMutex
}

func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		agents: make(map[string]AgentVersions),
	}
}

func (r *InMemoryRegistry) Store(name, version string, agent *AgentFile) error {
	v, err := ParseVersion(version)
	if err != nil {
		return fmt.Errorf("invalid version format: %v", err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.agents[name]; !exists {
		r.agents[name] = make(AgentVersions)
	}
	r.agents[name][v.String()] = agent
	return nil
}

func (r *InMemoryRegistry) Get(name, version string) (*AgentFile, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	versions, exists := r.agents[name]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", name)
	}

	// If version is empty or "latest", find the highest version
	if version == "" || version == "latest" {
		return r.getLatestVersion(name, versions)
	}

	v, err := ParseVersion(version)
	if err != nil {
		return nil, fmt.Errorf("invalid version format: %v", err)
	}

	agent, exists := versions[v.String()]
	if !exists {
		return nil, fmt.Errorf("version %s not found for agent %s", version, name)
	}

	return agent, nil
}

func (r *InMemoryRegistry) GetJSON(name, version string) ([]byte, error) {
	agent, err := r.Get(name, version)
	if err != nil {
		return nil, err
	}

	return json.Marshal(agent)
}

func (r *InMemoryRegistry) getLatestVersion(name string, versions AgentVersions) (*AgentFile, error) {
	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for agent %s", name)
	}

	var latest *Version
	var latestAgent *AgentFile

	for verStr, agent := range versions {
		ver, err := ParseVersion(verStr)
		if err != nil {
			continue
		}

		if latest == nil || ver.Compare(*latest) > 0 {
			latest = ver
			latestAgent = agent
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no valid versions found for agent %s", name)
	}

	return latestAgent, nil
}

func (r *InMemoryRegistry) Close() error {
	return nil
}
