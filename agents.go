// agents.go
package main

import (
	"context"
	"fmt"
	"log"
)

type Agent struct {
	id           AgentID
	broker       Broker
	executor     Executor
	metrics      *Metrics
	registry     *CapabilityRegistry
	taskTypes    []string
	skillsByType map[string][]string
	cancelFunc   context.CancelFunc
}

func NewAgent(
	id AgentID,
	b Broker,
	e Executor,
	m *Metrics,
	r *CapabilityRegistry,
	taskTypes []string,
	skillsByType map[string][]string,
) *Agent {
	return &Agent{
		id:           id,
		broker:       b,
		executor:     e,
		metrics:      m,
		registry:     r,
		taskTypes:    taskTypes,
		skillsByType: skillsByType,
	}
}

func (a *Agent) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel

	// Register agent's capabilities
	a.registry.Register(a.id, AgentCapability{
		AgentID:      a.id,
		TaskTypes:    a.taskTypes,
		SkillsByType: a.skillsByType,
		Resources: map[string]int{
			"cpu": 4,
			"gpu": 1,
		},
	})

	// Subscribe to agent-specific topic
	taskCh, err := a.broker.Subscribe(ctx, string(a.id))
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
		a.id, a.taskTypes, a.skillsByType)
	return nil
}

func (a *Agent) processTask(ctx context.Context, task *Task) {
	log.Printf("Agent %s starting work on task: %s (Type: %s, Required Skills: %v)",
		a.id, task.Title, task.Type, task.SkillsRequired)

	result, err := a.executor.Execute(ctx, task)
	if err != nil {
		log.Printf("Agent %s failed to execute task %s: %v", a.id, task.Title, err)
		a.metrics.RecordTaskError(task.Type, err)
		return
	}

	if result.Success {
		// Record completion with duration calculation
		a.metrics.RecordTaskComplete(task.Type, task.ID)
		log.Printf("Agent %s successfully completed task: %s", a.id, task.Title)
	} else {
		a.metrics.RecordTaskError(task.Type, fmt.Errorf("task completed unsuccessfully"))
		log.Printf("Agent %s task completed but unsuccessful: %s", a.id, task.Title)
	}
}

func (a *Agent) Stop(ctx context.Context) error {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	return nil
}
