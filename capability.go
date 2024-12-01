package main

import "sync"

type TaskCapability struct {
	Skills []string
}

type Capability struct {
	Name    string
	Version string
	Enabled bool
	// Map of task type to its required skills
	TaskCapabilities map[string]TaskCapability
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

func (r *CapabilityRegistry) FindMatchingAgents(taskType string, taskSkills []string) []AgentID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matchingAgents []AgentID
	for agentID, agentCap := range r.capabilities {
		// First check if agent supports this task type
		supportsTaskType := false
		for _, supportedType := range agentCap.TaskTypes {
			if supportedType == taskType {
				supportsTaskType = true
				break
			}
		}
		if !supportsTaskType {
			continue
		}

		// Then check if agent has all required skills for this task type
		agentSkills, hasTaskTypeSkills := agentCap.SkillsByType[taskType]
		if !hasTaskTypeSkills {
			continue
		}

		// Verify all required skills are present
		hasAllSkills := true
		for _, requiredSkill := range taskSkills {
			found := false
			for _, agentSkill := range agentSkills {
				if agentSkill == requiredSkill {
					found = true
					break
				}
			}
			if !found {
				hasAllSkills = false
				break
			}
		}

		if hasAllSkills {
			matchingAgents = append(matchingAgents, agentID)
		}
	}
	return matchingAgents
}
