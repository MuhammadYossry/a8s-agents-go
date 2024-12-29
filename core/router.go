// router.go
package core

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	"github.com/Relax-N-Tax/AgentNexus/internal/agents"
	"github.com/Relax-N-Tax/AgentNexus/metrics"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

type TaskRouter struct {
	registry     *capability.CapabilityRegistry
	broker       Broker
	metrics      *metrics.Metrics
	matcher      *capability.CapabilityMatcher
	routingAgent *agents.TaskRoutingAgent
}

func NewTaskRouter(registry *capability.CapabilityRegistry, broker Broker, metrics *metrics.Metrics, taskRoutingAgent *agents.TaskRoutingAgent) *TaskRouter {
	return &TaskRouter{
		registry:     registry,
		broker:       broker,
		metrics:      metrics,
		matcher:      capability.NewCapabilityMatcher(registry, capability.DefaultMatcherConfig()),
		routingAgent: taskRoutingAgent,
	}
}

// RouteTask attempts to find and assign a task to a capable agent
func (tr *TaskRouter) RouteTask(ctx context.Context, task *types.Task) error {
	// Find matching agent/action using the routing agent
	actionPlan, err := tr.routingAgent.FindMatchingAgent(ctx, task)
	if err != nil {
		log.Printf("router: finding matching agent failed: %v", err)
		tr.metrics.RecordRoutingFailure(task.Requirements, "agent_matching_failed")
		return fmt.Errorf("finding matching agent: %w", err)
	}

	// Check if we found a suitable match
	if actionPlan.Confidence == 0 || actionPlan.SelectedAction == "" {
		tr.metrics.RecordRoutingFailure(task.Requirements, "no_matching_agents")
		return fmt.Errorf("no capable agents found for task requirements")
	}

	// Update task status
	task.Status = types.TaskStatusPending
	task.UpdatedAt = time.Now()

	// Get agent ID from the action plan reasoning
	agentID := extractAgentIDFromReasoning(actionPlan.Reasoning)
	if agentID == "" {
		return fmt.Errorf("invalid action plan: missing agent ID")
	}

	// Publish task to the selected agent's topic
	if err := tr.broker.Publish(ctx, agentID, task); err != nil {
		tr.metrics.RecordRoutingFailure(task.Requirements, "publish_failed")
		return fmt.Errorf("failed to publish task to agent %s: %w", agentID, err)
	}

	tr.metrics.RecordRoutingSuccess(task.Requirements, agentID)
	return nil
}

// extractAgentIDFromReasoning extracts the agent ID from the action plan reasoning
func extractAgentIDFromReasoning(reasoning types.ActionPlanReasoning) string {
	for _, point := range reasoning.AlignmentPoints {
		// Look for the alignment point that contains the agent ID
		if len(point) > 6 && point[:6] == "Agent " {
			return point[6 : len(point)-9] // Remove "Agent " prefix and " selected" suffix
		}
	}
	return ""
}

// GetAgentLoad returns the number of pending tasks for an agent
func (tr *TaskRouter) GetAgentLoad(agentID types.AgentID) (int, error) {
	// TODO: Implement agent load tracking
	return 0, nil
}
