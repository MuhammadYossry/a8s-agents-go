package core

import (
	"strings"
	"sync"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

type TaskCapability struct {
	Skills []string
}

type CapabilityRegistry struct {
	mu           sync.RWMutex
	capabilities map[types.AgentID]types.AgentCapability
}

func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{
		capabilities: make(map[types.AgentID]types.AgentCapability),
	}
}

func (r *CapabilityRegistry) Register(agentID types.AgentID, cap types.AgentCapability) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.capabilities[agentID] = cap
}

func (r *CapabilityRegistry) FindMatchingAgents(task *types.Task) []types.AgentID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matchingAgents []types.AgentID
	for agentID, agentCap := range r.capabilities {
		if supportsTaskRequirements(agentCap, task.Requirements) {
			matchingAgents = append(matchingAgents, agentID)
		}
	}
	return matchingAgents
}

func supportsTaskRequirements(agentCap types.AgentCapability, req types.TaskRequirement) bool {
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

func matchesCapabilityPath(capPath []string, taskPath types.TaskPath) bool {
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

func (r *CapabilityRegistry) GetCapabilityByAgent(agentID types.AgentID) (types.AgentCapability, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cap, exists := r.capabilities[agentID]
	return cap, exists
}

// GetCapabilitiesBySkill returns all agent capabilities that have a specific skill
func (r *CapabilityRegistry) GetCapabilitiesBySkill(skill string) []types.AgentCapability {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []types.AgentCapability
	for _, cap := range r.capabilities {
		for _, capability := range cap.Capabilities {
			for _, skillPath := range capability.SkillPath {
				if strings.Contains(skillPath, skill) {
					result = append(result, cap)
					break
				}
			}
		}
	}
	return result
}

func (r *CapabilityRegistry) RegisterWorkflow(workflowID WorkFlowID, cap WorkFlowCapability) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agentCap := types.AgentCapability{
		AgentID:      types.AgentID(workflowID),
		Capabilities: cap.Capabilities,
		Resources:    cap.Resources,
	}
	r.capabilities[types.AgentID(workflowID)] = agentCap
}
