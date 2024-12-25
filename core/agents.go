// agents.go
package core

import (
	"context"
	"log"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/metrics"
	"github.com/Relax-N-Tax/AgentNexus/types"
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
	agentDefinition *AgentDefinition
	broker          Broker
	executor        Executor
	metrics         *metrics.Metrics
	registry        *CapabilityRegistry
	cancelFunc      context.CancelFunc
}

func NewAgent(
	def *AgentDefinition,
	broker Broker,
	executor Executor,
	metrics *metrics.Metrics,
	registry *CapabilityRegistry,
) *Agent {
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

	log.Printf("Agent %s started with %d capabilities and %d actions",
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
	log.Printf("Agent %s processing task: %s (Required Skills: %v)",
		a.ID, task.Title, task.Requirements.SkillPath)

	result, err := a.executor.Execute(ctx, task)
	if err != nil {
		log.Printf("Agent %s failed to execute task %s: %v", a.ID, task.Title, err)
		// a.metrics.RecordTaskError(task.Type, err)
	}

	if result.Success {
		// a.metrics.RecordTaskComplete(task.Type, task.ID)
		log.Printf("Agent %s successfully completed task: %s", a.ID, task.Title)
	} else {
		// a.metrics.RecordTaskError(task.Type, fmt.Errorf("task completed unsuccessfully"))
		log.Printf("Agent %s task completed but unsuccessful: %s", a.ID, task.Title)
	}
	return &types.TaskResult{
		TaskID:     task.ID,
		Success:    true,
		FinishedAt: time.Now(),
	}, nil
}

func (a *Agent) GetCapabilities() []types.AgentCapability {
	return a.registry.GetCapabilitiesBySkill(string(a.ID))
}
