// router.go
package core

import (
	"context"
	"fmt"
	"log"

	"github.com/MuhammadYossry/a8s-agents-go/capability"
	"github.com/MuhammadYossry/a8s-agents-go/types"
)

type TaskRouter struct {
	registry     *capability.CapabilityRegistry
	broker       types.Broker
	metrics      types.MetricsCollector
	routingAgent types.TaskRoutingAgent
}

func NewTaskRouter(
	registry *capability.CapabilityRegistry,
	broker types.Broker,
	metrics types.MetricsCollector,
	routingAgent types.TaskRoutingAgent,
) types.TaskRouter {
	return &TaskRouter{
		registry:     registry,
		broker:       broker,
		metrics:      metrics,
		routingAgent: routingAgent,
	}
}

// RouteTask attempts to find and assign a task to a capable agent
func (tr *TaskRouter) RouteTask(ctx context.Context, task *types.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	actionPlan, err := tr.routingAgent.FindMatchingAgent(ctx, task)
	if err != nil {
		return fmt.Errorf("finding matching agent: %w", err)
	}

	log.Printf("Routing task %s with confidence %.2f", task.ID, actionPlan.Confidence)

	// Update to include context in Publish call
	if err := tr.broker.Publish(ctx, string(task.ID), task); err != nil {
		return fmt.Errorf("publishing task: %w", err)
	}

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
