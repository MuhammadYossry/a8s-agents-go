// orchestrator/orchestrator.go
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	"github.com/Relax-N-Tax/AgentNexus/core"
	"github.com/Relax-N-Tax/AgentNexus/definationloader"
	"github.com/Relax-N-Tax/AgentNexus/hub"
	"github.com/Relax-N-Tax/AgentNexus/internal/agents"
	"github.com/Relax-N-Tax/AgentNexus/metrics"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

// Orchestrator manages the lifecycle and coordination of all agents
type Orchestrator struct {
	config      Config
	broker      types.Broker
	metrics     types.MetricsCollector
	registry    *capability.CapabilityRegistry
	hubRegistry hub.DefinationRegistry

	agents       []types.Agenter
	agentFactory types.AgentFactory

	mu sync.RWMutex
}

type Config struct {
	Agents         []types.AgentDefConfig // List of agent references (e.g., "myagent:1.0")
	InternalConfig types.InternalAgentConfig
}

func New(cfg Config) (*Orchestrator, error) {
	broker := core.NewPubSub()
	metrics := metrics.NewMetrics()
	registry := capability.GetCapabilityRegistry()
	factory := agents.NewAgentFactory()

	hubRegistry, err := hub.GetSQLiteRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to create SQLite registry: %v", err)
	}

	return &Orchestrator{
		config:       cfg,
		broker:       broker,
		metrics:      metrics,
		registry:     registry,
		agentFactory: factory,
		hubRegistry:  hubRegistry,
	}, nil
}

// Start initializes and starts all components
func (o *Orchestrator) Start(ctx context.Context) (context.Context, error) {
	agentsLoader := definationloader.NewAgentLoader(o.broker, o.metrics, o.registry, o.agentFactory, o.hubRegistry)

	var agentDefs []types.AgentDefinition
	for _, agentConfig := range o.config.Agents {
		if agentConfig.Version == "" {
			agentConfig.Version = "latest"
		}

		agentDef, err := o.hubRegistry.GetAgentDef(agentConfig.Name, agentConfig.Version)
		if err != nil {
			log.Printf("WARNING: %v\n", err)
		} else {
			agentDefs = append(agentDefs, *agentDef)
		}
	}

	// Generate markdown documentation
	mdFormatter := definationloader.NewMarkdownFormatter()
	var markdown strings.Builder
	for _, def := range agentDefs {
		md, err := mdFormatter.MarkDownFromAgent(def)
		if err != nil {
			return nil, fmt.Errorf("failed to generate markdown for agent %s: %w", def.ID, err)
		}
		markdown.WriteString(md)
		markdown.WriteString("\n---\n")
	}

	ctx = context.WithValue(ctx, types.AgentsMarkDownKey, markdown.String())
	ctx = context.WithValue(ctx, types.RawAgentsDataKey, agentDefs)

	ctx, agents, err := agentsLoader.LoadAgents(ctx, agentDefs, o.config.InternalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load agents: %w", err)
	}

	o.mu.Lock()
	o.agents = agents
	o.mu.Unlock()

	for _, agent := range agents {
		if err := agent.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start agent: %w", err)
		}
	}

	return ctx, nil
}

// ProcessQuery handles a user query through the task extraction pipeline
func (o *Orchestrator) ProcessQuery(ctx context.Context, query string) (context.Context, error) {
	// First extract the task
	taskExtractor, err := o.agentFactory.GetTaskExtractionAgent(o.config.InternalConfig)
	if err != nil {
		return nil, fmt.Errorf("orch: failed to initialize task extractor agent: %w", err)
	}
	ctx, err = taskExtractor.ExtractTaskWithRetry(ctx, query)
	if err != nil {
		return ctx, fmt.Errorf("orch: task extraction failed: %w", err)
	}

	return ctx, nil
}

// ExecuteTasks executes the provided agent-assigned tasks through the agent pipeline
func (o *Orchestrator) ExecuteTasks(ctx context.Context, tasks []*types.Task) error {
	if mdContent, ok := ctx.Value(types.AgentsMarkDownKey).(string); ok && mdContent != "" {
		log.Println("task-routing-agent: Found markdown content")

		// Create formatter from the markdown content
		mdFormatter := definationloader.NewMarkdownFormatter()
		parsedRoot := mdFormatter.ParseSections(mdContent)

		var formattedContent strings.Builder

		// Try to get each main section
		for title, section := range parsedRoot.Sections {
			log.Printf("task-routing-agent: Processing section: %s", title)
			switch title {
			case "Agent", "Capabilities", "Available Endpoints":
				log.Printf("task-routing-agent: Found %s section", title)
				formattedContent.WriteString(fmt.Sprintf("### %s\n", title))
				formattedContent.WriteString(section.Content + "\n\n")
			}
		}
	}
	log.Panicf("Not implemented")

	return nil
}

// ExecuteAgentAction executes the provided agent-assigned tasks through the agent pipeline
func (o *Orchestrator) ExecuteAgentAction(ctx context.Context, agentID types.AgentID) error {
	log.Panicf("Not implemented")

	return nil
}

// Shutdown gracefully shuts down all components
func (o *Orchestrator) Shutdown(ctx context.Context) error {
	o.mu.RLock()
	agents := o.agents
	o.mu.RUnlock()

	// Shutdown agents
	for _, agent := range agents {
		if err := agent.Stop(ctx); err != nil {
			log.Printf("failed to start agent with caps %v: %v", err, agent.GetCapabilities())

		}
	}

	// Close broker
	if err := o.broker.Close(); err != nil {
		return fmt.Errorf("error closing broker: %w", err)
	}

	// Print final metrics
	log.Println("\nTask Execution Metrics:")
	log.Println("----------------------")
	// Add your metrics logging here

	return nil
}
