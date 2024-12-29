// orchestrator/orchestrator.go
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	"github.com/Relax-N-Tax/AgentNexus/core"
	"github.com/Relax-N-Tax/AgentNexus/internal/agents"
	"github.com/Relax-N-Tax/AgentNexus/metrics"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

// Orchestrator manages the lifecycle and coordination of all agents
type Orchestrator struct {
	config   Config
	broker   core.Broker
	metrics  *metrics.Metrics
	registry *capability.CapabilityRegistry
	router   *core.TaskRouter

	agents    []*core.Agent
	extractor *agents.TaskExtractionAgent

	mu sync.RWMutex
}

// Config holds orchestrator initialization options
type Config struct {
	// I want to replace AgentConfigPath with
	AgentsConfigPath   string
	AgentsConfigMDPath string
	InternalConfig     types.InternalAgentConfig
}

// New creates a new Orchestrator instance
func New(cfg Config) (*Orchestrator, error) {
	broker := core.NewPubSub()
	metrics := metrics.NewMetrics()
	registry := capability.GetCapabilityRegistry()
	taskRoutingAgent, err := agents.GetTaskRoutingAgent(context.Background(), cfg.InternalConfig)
	if err != nil {
		log.Fatal(err)
	}
	router := core.NewTaskRouter(registry, broker, metrics, taskRoutingAgent)
	extractor, err := agents.GetTaskExtractionAgent(context.Background(), cfg.InternalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task extractor: %w", err)
	}

	return &Orchestrator{
		config:    cfg,
		broker:    broker,
		metrics:   metrics,
		registry:  registry,
		router:    router,
		extractor: extractor,
	}, nil
}

// Start initializes and starts all components
func (o *Orchestrator) Start(ctx context.Context) error {
	loader := core.NewAgentLoader(o.broker, o.metrics, o.registry)

	ctx, agents, err := loader.LoadAgents(ctx, o.config.AgentsConfigPath, o.config.InternalConfig)
	if err != nil {
		return fmt.Errorf("failed to load agents.json: %w", err)
	}

	o.mu.Lock()
	o.agents = agents
	o.mu.Unlock()

	// Start all agents
	for _, agent := range agents {
		if err := agent.Start(ctx); err != nil {
			return fmt.Errorf("failed to start agent %s: %w", agent.ID, err)
		}
	}

	return nil
}

// ProcessQuery handles a user query through the task extraction pipeline
func (o *Orchestrator) ProcessQuery(ctx context.Context, query string) (context.Context, error) {
	// First extract the task
	ctx, err := o.extractor.ExtractTaskWithRetry(ctx, query)
	if err != nil {
		return ctx, fmt.Errorf("task extraction failed: %w", err)
	}

	// Add raw agents data to context
	agentsData, err := o.loadRawAgentsData()
	if err != nil {
		return ctx, fmt.Errorf("failed to load agents data: %w", err)
	}
	ctx = context.WithValue(ctx, types.RawAgentsDataKey, agentsData)

	// Add markdown agents data to context  // <- New section
	markdownData, err := o.loadMarkdownAgentsData() // <- New
	if err != nil {
		return ctx, fmt.Errorf("failed to load markdown agents data: %w", err)
	}
	ctx = context.WithValue(ctx, types.AgentsMarkDownKey, markdownData) // <- New

	return ctx, nil
}

func (o *Orchestrator) loadMarkdownAgentsData() (string, error) {
	// Load and return agents config data as string
	data, err := os.ReadFile(o.config.AgentsConfigMDPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
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
			log.Printf("Error stopping agent %s: %v", agent.ID, err)
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
