package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	internal_agents "github.com/Relax-N-Tax/AgentNexus/internal/agents"
	"github.com/Relax-N-Tax/AgentNexus/metrics"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

type AgentLoader struct {
	broker   Broker
	metrics  *metrics.Metrics
	registry *capability.CapabilityRegistry
}

func NewAgentLoader(broker Broker, metrics *metrics.Metrics, registry *capability.CapabilityRegistry) *AgentLoader {
	return &AgentLoader{
		broker:   broker,
		metrics:  metrics,
		registry: registry,
	}
}

func (l *AgentLoader) LoadAgents(ctx context.Context, filepath string, internalAgentConfig types.InternalAgentConfig) (context.Context, []*Agent, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return ctx, nil, fmt.Errorf("reading config file: %w", err)
	}
	updatedContext := context.WithValue(ctx, types.RawAgentsDataKey, data)

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return ctx, nil, fmt.Errorf("parsing config: %w", err)
	}

	agents := make([]*Agent, 0, len(config.Agents))
	// Initialize PayloadAgent first
	payloadAgent, err := internal_agents.GetPayloadAgent(ctx, internalAgentConfig)
	if err != nil {
		log.Fatal(err)
	}
	// result, err := json.Marshal(config.Agents)
	// if err != nil {
	// 	return ctx, nil, fmt.Errorf("failed to Marshak agent definition: %w", err)
	// }

	for _, def := range config.Agents {
		if err := l.validateAgentDefinition(&def); err != nil {
			return ctx, nil, fmt.Errorf("invalid agent definition %s: %w", def.ID, err)
		}

		taskExecutor := NewTaskExecutor(TaskExecutorConfig{
			AgentDefinition: &def,
			PayloadAgent:    payloadAgent,
			HTTPTimeout:     30 * time.Second,
		})
		agent := NewAgent(&def, l.broker, taskExecutor, l.metrics, l.registry)
		agents = append(agents, agent)
	}

	return updatedContext, agents, nil
}

func (l *AgentLoader) validateAgentDefinition(def *AgentDefinition) error {
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
