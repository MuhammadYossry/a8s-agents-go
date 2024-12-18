package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type AgentLoader struct {
	broker   Broker
	metrics  *Metrics
	registry *CapabilityRegistry
}

func NewAgentLoader(broker Broker, metrics *Metrics, registry *CapabilityRegistry) *AgentLoader {
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
	for _, def := range config.Agents {
		if err := l.validateAgentDefinition(&def); err != nil {
			return nil, fmt.Errorf("invalid agent definition %s: %w", def.ID, err)
		}

		taskExecutor := NewTaskExecutor(&def)
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
		if err := l.validateAction(action); err != nil {
			return fmt.Errorf("invalid action %s: %w", action.Name, err)
		}
	}
	return nil
}

func (l *AgentLoader) validateAction(action Action) error {
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
