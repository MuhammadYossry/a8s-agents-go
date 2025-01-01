package definationloader

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	"github.com/Relax-N-Tax/AgentNexus/core"
	"github.com/Relax-N-Tax/AgentNexus/hub"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

type AgentLoader struct {
	broker       types.Broker
	metrics      types.MetricsCollector
	registry     *capability.CapabilityRegistry
	agentFactory types.AgentFactory
	hubRegistry  hub.DefinationRegistry
}

func NewAgentLoader(
	broker types.Broker,
	metrics types.MetricsCollector,
	registry *capability.CapabilityRegistry,
	agentFactory types.AgentFactory,
	hubRegistry hub.DefinationRegistry,
) *AgentLoader {
	return &AgentLoader{
		broker:       broker,
		metrics:      metrics,
		registry:     registry,
		agentFactory: agentFactory,
		hubRegistry:  hubRegistry,
	}
}

func (l *AgentLoader) LoadAgentsFromConfig(ctx context.Context, agentRefs []string, internalAgentConfig types.InternalAgentConfig) (context.Context, []types.Agenter, error) {
	agents := make([]types.Agenter, 0, len(agentRefs))
	agentDefs := make([]types.AgentDefinition, 0, len(agentRefs))

	// Get payload agent once outside the loop since it's reused
	payloadAgent, err := l.agentFactory.GetPayloadAgent(ctx, internalAgentConfig)
	if err != nil {
		return ctx, nil, fmt.Errorf("getting payload agent: %w", err)
	}

	// Process each agent reference
	for _, ref := range agentRefs {
		// Parse name and version from reference (e.g. "myagent:v1.0")
		name, version := parseAgentRef(ref)
		if name == "" {
			return ctx, nil, fmt.Errorf("invalid agent reference: empty name")
		}

		// Get agent file from registry
		agentFile, err := l.hubRegistry.Get(name, version)
		if err != nil {
			return ctx, nil, fmt.Errorf("failed to get agent %s: %w", ref, err)
		}

		// Parse agent definition from Content field
		var def types.AgentDefinition
		if err := json.Unmarshal([]byte(agentFile.Content), &def); err != nil {
			return ctx, nil, fmt.Errorf("failed to parse agent definition for %s: %w", ref, err)
		}

		// Basic validation
		if def.ID == "" || def.BaseURL == "" {
			return ctx, nil, fmt.Errorf("invalid agent definition %s: missing required fields", ref)
		}

		agentDefs = append(agentDefs, def)

		// Create task executor and agent
		taskExecutor := core.NewTaskExecutor(types.TaskExecutorConfig{
			AgentDefinition: &def,
			PayloadAgent:    payloadAgent,
			HTTPTimeout:     30 * time.Second,
		})

		agent := core.NewAgent(&def, l.broker, taskExecutor, l.metrics, l.registry)
		agents = append(agents, agent)
	}

	// Store raw agent data in context
	rawData, err := json.Marshal(struct {
		Agents []types.AgentDefinition `json:"agents"`
	}{
		Agents: agentDefs,
	})
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to marshal agent definitions: %w", err)
	}

	updatedContext := context.WithValue(ctx, types.RawAgentsDataKey, string(rawData))

	return updatedContext, agents, nil
}

func parseAgentRef(ref string) (name string, version string) {
	parts := strings.Split(ref, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
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
			return ctx, nil, fmt.Errorf("1 invalid agent definition %s: %w", def.ID, err)
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
