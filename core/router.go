// router.go
package core

import (
	"context"
	"fmt"
	"log"

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
	// Use capability matcher to find matching agents/workflows
	taskExtractionResult := ctx.Value(types.TaskExtractionResultKey).(string)
	log.Printf("router: received Task: %s\n", taskExtractionResult)
	rawAgentsData := ctx.Value(types.RawAgentsDataKey)
	if rawAgentsData == nil {
		return fmt.Errorf("missing raw agents data in context")
	}
	agentsDataStr, ok := rawAgentsData.(string)
	if !ok {
		return fmt.Errorf("invalid raw agents data type")
	}
	log.Printf("router: received rawAgentsData: %s\n", agentsDataStr)
	// TO THE LLM READ THAT
	// OUTPUT Router: Extracted Task: {
	// 	"id": "task001",
	// 	"title": "Build a REST API with Django Rest Framework",
	// 	"description": "Develop a RESTful API using Python and the Django Rest Framework to handle HTTP requests and responses efficiently.",
	// 	"requirements": {
	// 		"skillPath": ["Web Development", "Back-end Development", "Python", "Django Rest Framework"],
	// 		"action": "Build",
	// 		"parameters": {
	// 			"language": "Python",
	// 			"framework": "Django Rest Framework",
	// 			"apiType": "REST"
	// 		}
	// 	}
	// }

	// actionPlan, err := tr.routingAgent.PlanAction(ctx, task, e.AgentDef.Actions)
	// if err != nil {
	// 	log.Printf("router: planning agent/action failed: %w", err)
	// }
	// matches := tr.matcher.FindMatchingAgents(task)
	// if len(matches) == 0 {
	// 	err := fmt.Errorf("no capable agents found for task requirements: path=%v, action=%s",
	// 		task.Requirements.SkillPath, task.Requirements.Action)
	// 	tr.metrics.RecordRoutingFailure(task.Requirements, "no_matching_agents")
	// 	return err
	// }
	return fmt.Errorf("router: planning agent is still WIP")

	// // Select the highest scoring match
	// selectedMatch := matches[0]
	// task.Status = types.TaskStatusPending
	// task.UpdatedAt = time.Now()

	// topic := string(selectedMatch.AgentID)
	// if err := tr.broker.Publish(ctx, topic, task); err != nil {
	// 	tr.metrics.RecordRoutingFailure(task.Requirements, "publish_failed")
	// 	return fmt.Errorf("failed to publish task to agent %s: %w", selectedMatch.AgentID, err)
	// }

	// tr.metrics.RecordRoutingSuccess(task.Requirements, string(selectedMatch.AgentID))
	// return nil
}

// GetAgentLoad returns the number of pending tasks for an agent
// This could be used for load balancing in future implementations
func (tr *TaskRouter) GetAgentLoad(agentID types.AgentID) (int, error) {
	// TODO: Implement agent load tracking
	return 0, nil
}
