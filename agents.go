package main

import (
	"context"
	"log"
)

type Agent struct {
	id           AgentID
	broker       Broker
	executor     Executor
	metrics      *Metrics
	registry     *CapabilityRegistry
	capabilities []string
	cancelFunc   context.CancelFunc
}

func NewAgent(id AgentID, b Broker, e Executor, m *Metrics, r *CapabilityRegistry, capabilities []string) *Agent {
	return &Agent{
		id:           id,
		broker:       b,
		executor:     e,
		metrics:      m,
		registry:     r,
		capabilities: capabilities,
	}
}

func (a *Agent) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel

	// Register agent's capabilities
	a.registry.Register(a.id, AgentCapability{
		AgentID:   a.id,
		TaskTypes: a.capabilities,
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

	log.Printf("Agent %s started with capabilities: %v", a.id, a.registry.capabilities[a.id].TaskTypes)
	return nil
}

func (a *Agent) processTask(ctx context.Context, task *Task) {
	log.Printf("Agent %s starting work on task: %s", a.id, task.Title)

	result, err := a.executor.Execute(ctx, task)
	if err != nil {
		log.Printf("Agent %s failed to execute task %s: %v", a.id, task.Title, err)
		a.metrics.RecordTaskError(task.Type, err)
		return
	}

	if result.Success {
		log.Printf("Agent %s successfully completed task: %s", a.id, task.Title)
		a.metrics.RecordTaskComplete(task.Type)
	}
}

func (a *Agent) Stop(ctx context.Context) error {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	return a.broker.Close()
}
