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

func (l *AgentLoader) LoadAgents(filepath string) ([]*Agent, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	agents := make([]*Agent, 0, len(config.Agents))
	// Initialize PayloadAgent first
	internalAgentConfig := types.InternalAgentConfig{
		LLMConfig: struct {
			BaseURL string
			APIKey  string
			Model   string
			Timeout time.Duration
		}{
			BaseURL: os.Getenv("RNT_OPENAI_URL"),
			APIKey:  os.Getenv("RNT_OPENAI_API_KEY"),
			Model:   "Qwen-2.5-72B-Chat",
			Timeout: 50 * time.Second,
		},
	}

	payloadAgent, err := internal_agents.GetPayloadAgent(context.Background(), internalAgentConfig)
	if err != nil {
		log.Fatal(err)
	}
	actionPlannerAgent, err := internal_agents.GetActionPlannerAgent(context.Background(), internalAgentConfig)
	if err != nil {
		log.Fatal(err)
	}
	for _, def := range config.Agents {
		if err := l.validateAgentDefinition(&def); err != nil {
			return nil, fmt.Errorf("invalid agent definition %s: %w", def.ID, err)
		}

		taskExecutor := NewTaskExecutor(TaskExecutorConfig{
			AgentDefinition:    &def,
			PayloadAgent:       payloadAgent,
			ActionPlannerAgent: actionPlannerAgent,
			HTTPTimeout:        30 * time.Second,
		})
		agent := NewAgent(&def, l.broker, taskExecutor, l.metrics, l.registry)
		agents = append(agents, agent)
	}

	return agents, nil
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
