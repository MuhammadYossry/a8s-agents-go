package definationloader

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	"github.com/Relax-N-Tax/AgentNexus/core"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

type AgentLoader struct {
	broker       types.Broker
	metrics      types.MetricsCollector
	registry     *capability.CapabilityRegistry
	agentFactory types.AgentFactory
}

func NewAgentLoader(broker types.Broker, metrics types.MetricsCollector, registry *capability.CapabilityRegistry, agentFactory types.AgentFactory) *AgentLoader {
	return &AgentLoader{
		broker:       broker,
		metrics:      metrics,
		registry:     registry,
		agentFactory: agentFactory,
	}
}

func (l *AgentLoader) LoadAgents(ctx context.Context, filepath string, internalAgentConfig types.InternalAgentConfig) (context.Context, []types.Agenter, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return ctx, nil, fmt.Errorf("reading config file: %w", err)
	}

	var config types.AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return ctx, nil, fmt.Errorf("parsing config: %w", err)
	}

	updatedContext := context.WithValue(ctx, types.RawAgentsDataKey, string(data))
	agents := make([]types.Agenter, 0, len(config.Agents))

	payloadAgent, err := l.agentFactory.GetPayloadAgent(ctx, internalAgentConfig)
	if err != nil {
		return ctx, nil, fmt.Errorf("getting payload agent: %w", err)
	}

	for _, def := range config.Agents {
		if err := l.validateAgentDefinition(&def); err != nil {
			return ctx, nil, fmt.Errorf("invalid agent definition %s: %w", def.ID, err)
		}

		taskExecutor := core.NewTaskExecutor(types.TaskExecutorConfig{
			AgentDefinition: &def,
			PayloadAgent:    payloadAgent,
			HTTPTimeout:     30 * time.Second,
		})
		agent := core.NewAgent(&def, l.broker, taskExecutor, l.metrics, l.registry)
		agents = append(agents, agent)
	}

	return updatedContext, agents, nil
}

func (l *AgentLoader) validateAgentDefinition(def *types.AgentDefinition) error {
	if def.ID == "" {
		return fmt.Errorf("missing agent ID")
	}
	if def.BaseURL == "" {
		return fmt.Errorf("missing base URL")
	}
	for _, action := range def.Actions {
		// XXX: I should use def.BaseURL only if action.BaseURL is not provided
		action.BaseURL = def.BaseURL
		if err := l.validateAction(action); err != nil {
			return fmt.Errorf("invalid action %s: %w", action.Name, err)
		}
	}
	return nil
}

func (l *AgentLoader) validateAction(action types.Action) error {
	if action.Name == "" {
		return fmt.Errorf("missing action name")
	}
	if action.Path == "" {
		return fmt.Errorf("missing action path")
	}
	if action.Method == "" {
		return fmt.Errorf("missing HTTP method")
	}
	return nil
}
