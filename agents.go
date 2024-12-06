// agents.go
package main

import (
	"context"
	"fmt"
	"log"
)

const (
	AgentTypeInternal AgentType = "internal"
	AgentTypeExternal AgentType = "external"
)

type Agent struct {
	ID           AgentID
	Type         AgentType
	Description  string
	TaskTypes    []string
	SkillsByType map[string][]string
	// External agent specific fields
	BaseURL         string
	agentDefination *AgentDefinition
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
	ID           string              `json:"id"`
	Type         string              `json:"type"`
	Description  string              `json:"description"`
	BaseURL      string              `json:"baseURL"`
	TaskTypes    []string            `json:"taskTypes"`
	SkillsByType map[string][]string `json:"skillsByType"`
	Actions      []Action            `json:"actions"`
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
	id AgentID,
	b Broker,
	e Executor,
	m *Metrics,
	r *CapabilityRegistry,
	agentDefination *AgentDefinition,
) *Agent {
	return &Agent{
		ID:              id,
		broker:          b,
		executor:        e,
		metrics:         m,
		registry:        r,
		TaskTypes:       agentDefination.TaskTypes,
		SkillsByType:    agentDefination.SkillsByType,
		BaseURL:         agentDefination.BaseURL,
		agentDefination: agentDefination,
	}
}

func (a *Agent) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel

	// Register agent's capabilities
	a.registry.Register(a.ID, AgentCapability{
		AgentID:      a.ID,
		TaskTypes:    a.TaskTypes,
		SkillsByType: a.SkillsByType,
		Resources: map[string]int{
			"cpu": 4,
			"gpu": 1,
		},
	})

	// Subscribe to agent-specific topic
	taskCh, err := a.broker.Subscribe(ctx, string(a.ID))
	if err != nil {
		return err
	}

	// Start task processing
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

	log.Printf("Agent %s started with capabilities - Task Types: %v, Skills by Type: %v",
		a.ID, a.TaskTypes, a.SkillsByType)
	return nil
}

func (a *Agent) processTask(ctx context.Context, task *Task) {
	log.Printf("Agent %s starting work on task: %s (Type: %s, Required Skills: %v)",
		a.ID, task.Title, task.Type, task.SkillsRequired)

	result, err := a.executor.Execute(ctx, task)
	if err != nil {
		log.Printf("Agent %s failed to execute task %s: %v", a.ID, task.Title, err)
		a.metrics.RecordTaskError(task.Type, err)
		return
	}

	if result.Success {
		// Record completion with duration calculation
		a.metrics.RecordTaskComplete(task.Type, task.ID)
		log.Printf("Agent %s successfully completed task: %s", a.ID, task.Title)
	} else {
		a.metrics.RecordTaskError(task.Type, fmt.Errorf("task completed unsuccessfully"))
		log.Printf("Agent %s task completed but unsuccessful: %s", a.ID, task.Title)
	}
}

func (a *Agent) Stop(ctx context.Context) error {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	return nil
}
