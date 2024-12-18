package main

import (
	"strings"
	"sync"
)

type TaskCapability struct {
	Skills []string
}

type Capability struct {
	SkillPath []string               `json:"skillPath"`
	Level     string                 `json:"level"`
	Metadata  map[string]interface{} `json:"metadata"`
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

func (r *CapabilityRegistry) FindMatchingAgents(task *Task) []AgentID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matchingAgents []AgentID
	for agentID, agentCap := range r.capabilities {
		if supportsTaskRequirements(agentCap, task.Requirements) {
			matchingAgents = append(matchingAgents, agentID)
		}
	}
	return matchingAgents
}

func supportsTaskRequirements(agentCap AgentCapability, req TaskRequirement) bool {
	// Check if agent has matching capability path
	hasMatchingPath := false
	for _, cap := range agentCap.Capabilities {
		if matchesCapabilityPath(cap.SkillPath, req.SkillPath) {
			hasMatchingPath = true
			break
		}
	}
	if !hasMatchingPath {
		return false
	}

	// Check if agent has matching action
	for _, action := range agentCap.Actions {
		if action.Name == req.Action {
			return true
		}
	}
	return false
}

func matchesCapabilityPath(capPath []string, taskPath TaskPath) bool {
	if len(taskPath) > len(capPath) {
		return false
	}

	// Check if capability path contains all elements of task path in order
	for i, pathElement := range taskPath {
		if !strings.EqualFold(capPath[i], pathElement) {
			return false
		}
	}
	return true
}

func (r *CapabilityRegistry) RegisterWorkflow(workflowID WorkFlowID, cap WorkFlowCapability) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agentCap := AgentCapability{
		AgentID:      AgentID(workflowID),
		Capabilities: cap.Capabilities,
		Resources:    cap.Resources,
	}
	r.capabilities[AgentID(workflowID)] = agentCap
}
