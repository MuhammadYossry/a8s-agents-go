// core/agent_orchestrator.go
package core

import (
	"context"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	"github.com/Relax-N-Tax/AgentNexus/internal/agents"
	"github.com/Relax-N-Tax/AgentNexus/metrics"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

type AgentOrchestrator struct {
	taskExtractor *agents.TaskExtractionAgent
	loader        *AgentLoader
	router        *TaskRouter
}

func NewAgentOrchestrator(config types.InternalAgentConfig, broker Broker, metrics *metrics.Metrics, registry *capability.CapabilityRegistry) (*AgentOrchestrator, error) {
	taskExtractor, err := agents.GetTaskExtractionAgent(context.Background(), config)
	if err != nil {
		return nil, err
	}

	loader := NewAgentLoader(broker, metrics, registry)
	router := NewTaskRouter(registry, broker, metrics)

	return &AgentOrchestrator{
		taskExtractor: taskExtractor,
		loader:        loader,
		router:        router,
	}, nil
}

func (o *AgentOrchestrator) ProcessQuery(ctx context.Context, query string) (context.Context, error) {
	return o.taskExtractor.ExtractTaskWithRetry(ctx, query)
}

func (o *AgentOrchestrator) LoadAndStartAgents(ctx context.Context, configPath string) ([]*Agent, error) {
	agents, err := o.loader.LoadAgents(configPath)
	if err != nil {
		return nil, err
	}

	for _, agent := range agents {
		if err := agent.Start(ctx); err != nil {
			return agents, err
		}
	}

	return agents, nil
}
