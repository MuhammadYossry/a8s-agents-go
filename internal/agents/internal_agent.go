package agents

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/metrics"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

// InternalAgent implements the core.Agent interface
type InternalAgent struct {
	id           types.AgentID
	agentType    types.AgentType
	description  string
	capabilities map[string]*InternalCapability
	actions      []types.Action
	memory       *Memory
	// llmClient    *LLMClient
	// promptMgr    *PromptManager
	metrics types.MetricsCollector
	mu      sync.RWMutex
}

func NewInternalAgent(id string, agentType types.AgentType, description string) *InternalAgent {
	return &InternalAgent{
		id:           types.AgentID(id),
		agentType:    agentType,
		description:  description,
		capabilities: make(map[string]*InternalCapability),
		memory:       NewMemory(1000),
		metrics:      metrics.NewMetrics().(*metrics.Metrics),
	}
}

func (a *InternalAgent) Start(ctx context.Context) error {
	log.Printf("Internal Agent %s started with %d capabilities",
		a.id, len(a.capabilities))

	a.memory.AddMemory("agent_event", map[string]interface{}{
		"event":              "start",
		"capabilities_count": len(a.capabilities),
	})
	return nil
}

func (a *InternalAgent) Stop(ctx context.Context) error {
	a.memory.AddMemory("agent_event", map[string]interface{}{
		"event": "stop",
	})
	return nil
}

func (a *InternalAgent) Execute(ctx context.Context, task *types.Task) (*types.TaskResult, error) {
	log.Printf("Internal Agent %s processing task: %s (Required Skills: %v)",
		a.id, task.Title, task.Requirements.SkillPath)

	a.memory.AddMemory("task_start", map[string]interface{}{
		"task_id": task.ID,
		"title":   task.Title,
		"skills":  task.Requirements.SkillPath,
	})

	a.mu.RLock()
	cap, exists := a.capabilities[task.Requirements.Action]
	a.mu.RUnlock()

	if !exists {
		err := fmt.Errorf("capability %s not found", task.Requirements.Action)
		a.memory.AddMemory("task_error", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      err.Error(),
			FinishedAt: time.Now(),
		}, nil
	}

	// Execute the capability
	result, err := cap.Execute(ctx, task)
	if err != nil {
		a.memory.AddMemory("task_error", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      err.Error(),
			FinishedAt: time.Now(),
		}, nil
	}

	a.memory.AddMemory("task_complete", map[string]interface{}{
		"task_id": task.ID,
		"result":  result,
	})

	return result, nil
}

func (a *InternalAgent) GetCapabilities() []types.AgentCapability {
	a.mu.RLock()
	defer a.mu.RUnlock()

	capabilities := make([]types.Capability, 0, len(a.capabilities))
	for _, cap := range a.capabilities {
		capabilities = append(capabilities, cap.Capability)
	}

	return []types.AgentCapability{{
		AgentID:      a.id,
		Capabilities: capabilities,
		Actions:      a.actions,
		Resources: map[string]int{
			"cpu": 1,
		},
	}}
}

func (a *InternalAgent) RegisterCapability(cap *InternalCapability) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if cap == nil {
		return fmt.Errorf("capability cannot be nil")
	}

	metadata := cap.GetMetadata()
	name, ok := metadata["name"].(string)
	if !ok {
		return fmt.Errorf("capability must have a name in metadata")
	}

	a.capabilities[name] = cap
	return nil
}

// ExecuteCapability executes a specific capability by name
func (a *InternalAgent) ExecuteCapability(ctx context.Context, name string, payload []byte) ([]byte, error) {
	a.mu.RLock()
	cap, exists := a.capabilities[name]
	a.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("capability %s not found", name)
	}

	// Create a task for the capability
	task := &types.Task{
		ID: fmt.Sprintf("task-%d", time.Now().UnixNano()),
		Requirements: types.TaskRequirement{
			Action:     name,
			SkillPath:  cap.GetSkillPath(),
			Parameters: make(map[string]interface{}),
		},
		Payload: payload,
	}

	result, err := cap.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	return result.Output, nil
}

// Extras
func (a *InternalAgent) GetRecentTaskHistory(limit int) []MemoryItem {
	return a.memory.GetRecentMemories("task_complete", limit)
}

func (a *InternalAgent) SearchTaskResults(taskID string) []MemoryItem {
	allResults := a.memory.SearchMemories("task_complete")
	var taskResults []MemoryItem

	for _, result := range allResults {
		if content, ok := result.Content.(map[string]interface{}); ok {
			if content["task_id"] == taskID {
				taskResults = append(taskResults, result)
			}
		}
	}

	return taskResults
}
