// router.go
package core

import (
	"context"
	"fmt"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	"github.com/Relax-N-Tax/AgentNexus/metrics"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

type TaskRouter struct {
	registry *capability.CapabilityRegistry
	broker   Broker
	metrics  *metrics.Metrics
	matcher  *capability.CapabilityMatcher
}

func NewTaskRouter(registry *capability.CapabilityRegistry, broker Broker, metrics *metrics.Metrics) *TaskRouter {
	return &TaskRouter{
		registry: registry,
		broker:   broker,
		metrics:  metrics,
		matcher:  capability.NewCapabilityMatcher(registry, capability.DefaultMatcherConfig()),
	}
}

// RouteTask attempts to find and assign a task to a capable agent
func (tr *TaskRouter) RouteTask(ctx context.Context, task *types.Task) error {
	// Use capability matcher to find matching agents/workflows
	matches := tr.matcher.FindMatchingAgents(task)
	if len(matches) == 0 {
		err := fmt.Errorf("no capable agents found for task requirements: path=%v, action=%s",
			task.Requirements.SkillPath, task.Requirements.Action)
		tr.metrics.RecordRoutingFailure(task.Requirements, "no_matching_agents")
		return err
	}

	// Select the highest scoring match
	selectedMatch := matches[0]
	task.Status = types.TaskStatusPending
	task.UpdatedAt = time.Now()

	topic := string(selectedMatch.AgentID)
	if err := tr.broker.Publish(ctx, topic, task); err != nil {
		tr.metrics.RecordRoutingFailure(task.Requirements, "publish_failed")
		return fmt.Errorf("failed to publish task to agent %s: %w", selectedMatch.AgentID, err)
	}

	tr.metrics.RecordRoutingSuccess(task.Requirements, string(selectedMatch.AgentID))
	return nil
}

// GetAgentLoad returns the number of pending tasks for an agent
// This could be used for load balancing in future implementations
func (tr *TaskRouter) GetAgentLoad(agentID types.AgentID) (int, error) {
	// TODO: Implement agent load tracking
	return 0, nil
}
