package main

import (
	"context"
	"fmt"
)

type TaskRouter struct {
	registry *CapabilityRegistry
	broker   Broker
}

func NewTaskRouter(registry *CapabilityRegistry, broker Broker) *TaskRouter {
	return &TaskRouter{
		registry: registry,
		broker:   broker,
	}
}

// In router.go
func (tr *TaskRouter) RouteTask(ctx context.Context, task *Task) error {
	// Find matching agents - use task.Capabilities instead of task.Type
	matchingAgents := tr.registry.FindMatchingAgents(task.Capabilities)
	if len(matchingAgents) == 0 {
		return fmt.Errorf("no capable agents found for task capabilities: %v", task.Capabilities)
	}

	// Route to first matching agent
	selectedAgent := matchingAgents[0]
	topic := string(selectedAgent) // Use agent ID as topic

	return tr.broker.Publish(ctx, topic, task)
}
