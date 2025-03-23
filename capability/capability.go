package capability

import (
	"strings"
	"sync"

	"github.com/MuhammadYossry/a8s-agents-go/types"
)

type TaskCapability struct {
	Skills []string
}

type CapabilityRegistry struct {
	mu           sync.RWMutex
	capabilities map[types.AgentID]types.AgentCapability
}

var (
	instance *CapabilityRegistry
	once     sync.Once
)

func GetCapabilityRegistry() *CapabilityRegistry {
	once.Do(func() {
		instance = &CapabilityRegistry{
			capabilities: make(map[types.AgentID]types.AgentCapability),
		}
	})
	return instance
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

// GetAllCapabilities returns all capabilities in the registry
func (r *CapabilityRegistry) GetAllCapabilities() []types.AgentCapability {
	r.mu.RLock()
	defer r.mu.RUnlock()

	caps := make([]types.AgentCapability, 0, len(r.capabilities))
	for _, cap := range r.capabilities {
		caps = append(caps, cap)
	}
	return caps
}

// GetTopLevelCapabilities returns a formatted string of L1 and L2 capabilities
func (r *CapabilityRegistry) GetTopLevelCapabilities() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Use maps to deduplicate paths
	l1Paths := make(map[string]struct{})
	l2Paths := make(map[string]struct{})

	for _, agentCap := range r.capabilities {
		for _, cap := range agentCap.Capabilities {
			if len(cap.SkillPath) >= 1 {
				l1Paths[cap.SkillPath[0]] = struct{}{}
			}
			if len(cap.SkillPath) >= 2 {
				l2Path := strings.Join(cap.SkillPath[:2], " --> ")
				l2Paths[l2Path] = struct{}{}
			}
		}
	}

	// Build formatted string
	var result strings.Builder

	// Add L1 paths
	for path := range l1Paths {
		result.WriteString("[")
		result.WriteString(path)
		result.WriteString("] ")
	}

	// Add L2 paths
	for path := range l2Paths {
		result.WriteString("[")
		result.WriteString(path)
		result.WriteString("] ")
	}

	return strings.TrimSpace(result.String())
}

func (r *CapabilityRegistry) RegisterWorkflow(workflowID types.WorkFlowID, cap types.WorkFlowCapability) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agentCap := types.AgentCapability{
		AgentID:      types.AgentID(workflowID),
		Capabilities: cap.Capabilities,
		Resources:    cap.Resources,
	}
	r.capabilities[types.AgentID(workflowID)] = agentCap
}
