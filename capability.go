package main

import "sync"

type Capability struct {
	Name      string
	Version   string
	Enabled   bool
	TaskTypes []string
}

type CapabilityRegistry struct {
	mu           sync.RWMutex
	capabilities map[AgentID]AgentCapability
}

func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{
		capabilities: make(map[AgentID]AgentCapability),
	}
}

func (r *CapabilityRegistry) Register(agentID AgentID, cap AgentCapability) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.capabilities[agentID] = cap
}

func (r *CapabilityRegistry) FindMatchingAgents(taskCapabilities []string) []AgentID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matchingAgents []AgentID
	for agentID, agentCap := range r.capabilities {
		// Check if agent has all required capabilities
		hasAllCapabilities := true
		for _, requiredCap := range taskCapabilities {
			found := false
			for _, agentCapability := range agentCap.TaskTypes {
				if agentCapability == requiredCap {
					found = true
					break
				}
			}
			if !found {
				hasAllCapabilities = false
				break
			}
		}
		if hasAllCapabilities {
			matchingAgents = append(matchingAgents, agentID)
		}
	}
	return matchingAgents
}
