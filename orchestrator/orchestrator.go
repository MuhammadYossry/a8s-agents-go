// orchestrator/orchestrator.go
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	"github.com/Relax-N-Tax/AgentNexus/core"
	"github.com/Relax-N-Tax/AgentNexus/definationloader"
	"github.com/Relax-N-Tax/AgentNexus/internal/agents"
	"github.com/Relax-N-Tax/AgentNexus/metrics"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

// Orchestrator manages the lifecycle and coordination of all agents
type Orchestrator struct {
	config   Config
	broker   types.Broker
	metrics  types.MetricsCollector
	registry *capability.CapabilityRegistry
	router   types.TaskRouter

	agents        []types.Agenter
	taskExtractor types.TaskExtractionAgent
	agentFactory  types.AgentFactory

	mu sync.RWMutex
}

// Config holds orchestrator initialization options
type Config struct {
	AgentsConfigPath string
	InternalConfig   types.InternalAgentConfig
}

func New(cfg Config) (*Orchestrator, error) {
	broker := core.NewPubSub()
	metrics := metrics.NewMetrics()
	registry := capability.GetCapabilityRegistry()
	factory := agents.NewAgentFactory() // Implement this in core package?
	// *errors* undefined: core.NewAgentFactorycompilerUndeclaredImportedName

	taskRoutingAgent, err := factory.GetTaskRoutingAgent(cfg.InternalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task routing agent: %w", err)
	}

	router := core.NewTaskRouter(registry, broker, metrics, taskRoutingAgent)

	taskExtractor, err := factory.GetTaskExtractionAgent(cfg.InternalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task extractor: %w", err)
	}

	return &Orchestrator{
		config:        cfg,
		broker:        broker,
		metrics:       metrics,
		registry:      registry,
		router:        router,
		taskExtractor: taskExtractor,
		agentFactory:  factory,
	}, nil
}

// Start initializes and starts all components
func (o *Orchestrator) Start(ctx context.Context) (context.Context, error) {
	agentsLoader := definationloader.NewAgentLoader(o.broker, o.metrics, o.registry, o.agentFactory)
	ctx, agents, err := agentsLoader.LoadAgents(ctx, o.config.AgentsConfigPath, o.config.InternalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load agents.json: %w", err)
	}

	// Load and parse config for markdown generation
	var config types.AgentConfig
	agentsData, err := os.ReadFile(o.config.AgentsConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents config file: %w", err)
	}

	if err := json.Unmarshal(agentsData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse agents config: %w", err)
	}

	// Generate markdown documentation
	mdFormatter := definationloader.NewMarkdownFormatter()
	markdown, err := mdFormatter.MarkDownFromConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("generating markdown: %w", err)
	}

	// Parse sections after generating markdown
	mdFormatter.ParseSections(markdown)

	// Store markdown content in context
	ctx = context.WithValue(ctx, types.AgentsMarkDownKey, markdown)
	ctx = context.WithValue(ctx, types.AgentsMDFormatterKey, mdFormatter)

	ctx = context.WithValue(ctx, types.RawAgentsDataKey, string(agentsData))

	o.mu.Lock()
	o.agents = agents
	o.mu.Unlock()

	// Start all agents
	for _, agent := range agents {
		if err := agent.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start agent with caps %w: %v", err, agent.GetCapabilities())
		}
	}

	return ctx, nil
}

// ProcessQuery handles a user query through the task extraction pipeline
func (o *Orchestrator) ProcessQuery(ctx context.Context, query string) (context.Context, error) {
	// First extract the task
	ctx, err := o.taskExtractor.ExtractTaskWithRetry(ctx, query)
	if err != nil {
		return ctx, fmt.Errorf("task extraction failed: %w", err)
	}

	// Add raw agents data to context
	agentsData, err := o.loadRawAgentsData()
	if err != nil {
		return ctx, fmt.Errorf("failed to load agents data: %w", err)
	}
	ctx = context.WithValue(ctx, types.RawAgentsDataKey, agentsData)

	// log markdown agents data in the context

	return ctx, nil
}

func (o *Orchestrator) loadRawAgentsData() (string, error) {
	// Load and return agents config data as string
	data, err := os.ReadFile(o.config.AgentsConfigPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ExecuteTasks executes the provided tasks through the agent pipeline
func (o *Orchestrator) ExecuteTasks(ctx context.Context, tasks []*types.Task) error {
	for _, task := range tasks {
		log.Printf("Routing task: %s (Type: %s, Required Skills: %v)",
			task.Title, task.ID, task.Requirements.SkillPath)

		if err := o.router.RouteTask(ctx, task); err != nil {
			log.Printf("Failed to route task: %v", err)
			continue
		}

		// Add delay between tasks for readable logs
		time.Sleep(50 * time.Second)
	}

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
			log.Printf("failed to start agent with caps %w: %v", err, agent.GetCapabilities())

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
