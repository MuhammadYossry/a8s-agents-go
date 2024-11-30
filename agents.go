package main

import (
	"context"
	"log"
)

type Agent struct {
	broker     Broker
	executor   Executor
	metrics    *Metrics
	registry   *CapabilityRegistry
	cancelFunc context.CancelFunc
}

func NewAgent(b Broker, e Executor, m *Metrics, r *CapabilityRegistry) *Agent {
	return &Agent{
		broker:   b,
		executor: e,
		metrics:  m,
		registry: r,
	}
}

func (a *Agent) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel

	// Register default capabilities
	a.registry.Register(Capability{
		Name:      "default",
		Version:   "1.0",
		Enabled:   true,
		TaskTypes: []string{"default"},
	})

	// Subscribe to tasks
	taskCh, err := a.broker.Subscribe(ctx, "tasks")
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

	log.Println("Agent started")
	return nil
}

func (a *Agent) processTask(ctx context.Context, task *Task) {
	result, err := a.executor.Execute(ctx, task)
	if err != nil {
		a.metrics.RecordTaskError(task.Type, err)
		return
	}

	if result.Success {
		a.metrics.RecordTaskComplete(task.Type)
	}
}

func (a *Agent) Stop(ctx context.Context) error {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	return a.broker.Close()
}
