// agents.go
package core

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MuhammadYossry/a8s-agents-go/capability"
	"github.com/MuhammadYossry/a8s-agents-go/types"
)

const (
	AgentTypeInternal types.AgentType = "internal"
	AgentTypeExternal types.AgentType = "external"
)

// Implment core.Agenter interface
type Agent struct {
	ID              types.AgentID
	Type            types.AgentType
	Description     string
	BaseURL         string
	agentDefinition *types.AgentDefinition
	broker          types.Broker
	executor        types.Executor
	metrics         types.MetricsCollector
	registry        *capability.CapabilityRegistry
	cancelFunc      context.CancelFunc
}

func NewAgent(
	def *types.AgentDefinition,
	broker types.Broker,
	executor types.Executor,
	metrics types.MetricsCollector,
	registry *capability.CapabilityRegistry,
) types.Agenter {
	return &Agent{
		ID:              types.AgentID(def.ID),
		Type:            types.AgentType(def.Type),
		Description:     def.Description,
		BaseURL:         def.BaseURL,
		agentDefinition: def,
		broker:          broker,
		executor:        executor,
		metrics:         metrics,
		registry:        registry,
	}
}

func (a *Agent) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel

	agentCap := types.AgentCapability{
		AgentID:      a.ID,
		Capabilities: a.agentDefinition.Capabilities,
		Actions:      a.agentDefinition.Actions,
		Resources: map[string]int{
			"cpu": 4,
			"gpu": 1,
		},
	}
	a.registry.Register(a.ID, agentCap)

	taskCh, err := a.broker.Subscribe(ctx, string(a.ID))
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case task := <-taskCh:
				if task == nil {
					return
				}
				go a.Execute(ctx, task)
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Printf("Agents: %s started with %d capabilities and %d actions",
		a.ID, len(a.agentDefinition.Capabilities), len(a.agentDefinition.Actions))
	return nil
}

func (a *Agent) Stop(ctx context.Context) error {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	return nil
}

func (a *Agent) Execute(ctx context.Context, task *types.Task) (*types.TaskResult, error) {
	if task == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}

	log.Printf("Agent %s processing task: %s (Required Skills: %v)",
		a.ID, task.Title, task.Requirements.SkillPath)

	// Execute the task with timeout
	taskCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	result, err := a.executor.Execute(taskCtx, task)

	// Handle execution error
	if err != nil {
		log.Printf("Agent %s failed to execute task %s: %v", a.ID, task.Title, err)
		// a.metrics.IncrementMetric("task_errors")

		// Return a proper error result instead of propagating the error
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      fmt.Sprintf("execution error: %v", err),
			FinishedAt: time.Now(),
		}, nil
	}

	// Handle nil result
	if result == nil {
		log.Printf("Agent %s received nil result for task %s", a.ID, task.Title)
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      "task execution returned nil result",
			FinishedAt: time.Now(),
		}, nil
	}

	// Handle task result
	if result.Success {
		log.Printf("Agent %s successfully completed task: %s", a.ID, task.Title)
		// a.metrics.IncrementMetric("tasks_completed")
	} else {
		log.Printf("Agent %s task completed but unsuccessful: %s - Error: %s",
			a.ID, task.Title, result.Error)
		// a.metrics.IncrementMetric("task_failures")
	}

	// Always return the actual result from the executor
	return result, nil
}

func (a *Agent) GetCapabilities() []types.AgentCapability {
	return a.registry.GetCapabilitiesBySkill(string(a.ID))
}
