// router.go
package main

import (
	"context"
	"fmt"
	"time"
)

type TaskRouter struct {
	registry *CapabilityRegistry
	broker   Broker
	metrics  *Metrics
}

func NewTaskRouter(registry *CapabilityRegistry, broker Broker, metrics *Metrics) *TaskRouter {
	return &TaskRouter{
		registry: registry,
		broker:   broker,
		metrics:  metrics,
	}
}

// RouteTask attempts to find and assign a task to a capable agent
func (tr *TaskRouter) RouteTask(ctx context.Context, task *Task) error {
	// Find matching agents based on task type and required skills
	matchingAgents := tr.registry.FindMatchingAgents(task.Type, task.SkillsRequired)
	if len(matchingAgents) == 0 {
		err := fmt.Errorf("no capable agents found for task type: %s, required skills: %v",
			task.Type, task.SkillsRequired)
		tr.metrics.RecordRoutingFailure(task.Type, "no_matching_agents")
		return err
	}

	// For now, use a simple strategy: select the first matching agent
	// TODO: Implement more sophisticated agent selection strategies:
	// - Round robin
	// - Load balanced
	// - Resource availability based
	selectedAgent := matchingAgents[0]

	// Update task status before publishing
	task.Status = TaskStatusPending
	task.UpdatedAt = time.Now()

	// Use agent ID as the topic for routing
	topic := string(selectedAgent)
	if err := tr.broker.Publish(ctx, topic, task); err != nil {
		tr.metrics.RecordRoutingFailure(task.Type, "publish_failed")
		return fmt.Errorf("failed to publish task to agent %s: %w", selectedAgent, err)
	}

	tr.metrics.RecordRoutingSuccess(task.Type, string(selectedAgent))
	return nil
}

// GetAgentLoad returns the number of pending tasks for an agent
// This could be used for load balancing in future implementations
func (tr *TaskRouter) GetAgentLoad(agentID AgentID) (int, error) {
	// TODO: Implement agent load tracking
	return 0, nil
}
