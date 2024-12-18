// agents.go
package main

import (
	"context"
	"log"
)

const (
	AgentTypeInternal AgentType = "internal"
	AgentTypeExternal AgentType = "external"
)

type Agent struct {
	ID              AgentID
	Type            AgentType
	Description     string
	BaseURL         string
	agentDefinition *AgentDefinition
	broker          Broker
	executor        Executor
	metrics         *Metrics
	registry        *CapabilityRegistry
	cancelFunc      context.CancelFunc
}

type AgentConfig struct {
	Agents []AgentDefinition `json:"agents"`
}

type AgentDefinition struct {
	ID           string       `json:"id"`
	Type         string       `json:"type"`
	Description  string       `json:"description"`
	BaseURL      string       `json:"baseURL"`
	Capabilities []Capability `json:"capabilities"`
	Actions      []Action     `json:"actions"`
}

type Action struct {
	Name         string       `json:"name"`
	Path         string       `json:"path"`
	Method       string       `json:"method"`
	InputSchema  SchemaConfig `json:"inputSchema"`
	OutputSchema SchemaConfig `json:"outputSchema"`
}

type SchemaConfig struct {
	Type       string              `json:"type"`
	Required   []string            `json:"required,omitempty"`
	Fields     []string            `json:"fields,omitempty"`
	Properties map[string]Property `json:"properties,omitempty"`
}

type Property struct {
	Type    string   `json:"type"`
	Formats []string `json:"formats,omitempty"`
	MaxSize string   `json:"max_size,omitempty"`
	Enum    []string `json:"enum,omitempty"`
	Default string   `json:"default,omitempty"`
}

func NewAgent(
	def *AgentDefinition,
	broker Broker,
	executor Executor,
	metrics *Metrics,
	registry *CapabilityRegistry,
) *Agent {
	return &Agent{
		ID:              AgentID(def.ID),
		Type:            AgentType(def.Type),
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

	agentCap := AgentCapability{
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
				go a.processTask(ctx, task)
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Printf("Agent %s started with %d capabilities and %d actions",
		a.ID, len(a.agentDefinition.Capabilities), len(a.agentDefinition.Actions))
	return nil
}

func (a *Agent) processTask(ctx context.Context, task *Task) {
	log.Printf("Agent %s processing task: %s (Required Skills: %v)",
		a.ID, task.Title, task.Requirements.SkillPath)

	result, err := a.executor.Execute(ctx, task)
	if err != nil {
		log.Printf("Agent %s failed to execute task %s: %v", a.ID, task.Title, err)
		// a.metrics.RecordTaskError(task.Type, err)
		return
	}

	if result.Success {
		// a.metrics.RecordTaskComplete(task.Type, task.ID)
		log.Printf("Agent %s successfully completed task: %s", a.ID, task.Title)
	} else {
		// a.metrics.RecordTaskError(task.Type, fmt.Errorf("task completed unsuccessfully"))
		log.Printf("Agent %s task completed but unsuccessful: %s", a.ID, task.Title)
	}
}

func (a *Agent) Stop(ctx context.Context) error {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	return nil
}
