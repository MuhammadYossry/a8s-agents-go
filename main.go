package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	"github.com/Relax-N-Tax/AgentNexus/core"
	"github.com/Relax-N-Tax/AgentNexus/metrics"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker := core.NewPubSub()
	metrics := metrics.NewMetrics()
	registry := capability.GetCapabilityRegistry()
	router := core.NewTaskRouter(registry, broker, metrics)
	config := types.InternalAgentConfig{
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

	orchestrator, err := core.NewAgentOrchestrator(config, broker, metrics, registry)
	if err != nil {
		log.Fatal(err)
	}

	ctx, err = orchestrator.ProcessQuery(ctx, "Build a REST API using Python Django Rest")
	if err != nil {
		log.Fatal(err)
	}

	result := ctx.Value(types.TaskExtractionResultKey).(string)
	log.Printf("Extracted Task: %s\n", result)

	agents, err := orchestrator.LoadAndStartAgents(ctx, "examples/agents_generated.json")
	if err != nil {
		log.Fatalf("Failed to load agents: %v", err)
	}

	// Start agents
	for _, agent := range agents {
		if err := agent.Start(ctx); err != nil {
			log.Fatalf("Failed to start agent %s: %v", agent.ID, err)
		}
	}

	tasks := []*types.Task{
		{
			ID:          "task1",
			Title:       "Create a REST API endpoint",
			Description: "Create a REST API endpoint using Python",
			Requirements: types.TaskRequirement{
				SkillPath: types.TaskPath{"Development", "Backend", "Python", "CodeGeneration"},
				Action:    "generateCode",
				Parameters: map[string]interface{}{
					"framework": "FastAPI",
				},
			},
			Status:    types.TaskStatusPending,
			CreatedAt: time.Now(),
		},
		{
			ID:          "task2",
			Title:       "Deploy API Preview",
			Description: "Deploy preview environment for Python API",
			Requirements: types.TaskRequirement{
				SkillPath: types.TaskPath{"Development", "Deployment", "Python"},
				Action:    "deployPreview",
				Parameters: map[string]interface{}{
					"branchId":  "feature-123",
					"isPrivate": true,
					"environmentVars": map[string]string{
						"DEBUG":   "true",
						"API_KEY": "preview-key",
					},
				},
			},
			Status:    types.TaskStatusPending,
			CreatedAt: time.Now(),
		},
		// {
		// 	ID:          "task3",
		// 	Title:       "Improve Code Quality",
		// 	Description: "Refactor and optimize Python codebase",
		// 	Requirements: types.TaskRequirement{
		// 		SkillPath: types.TaskPath{"Development", "Backend", "Python", "CodeGeneration"},
		// 		Action:    "improveCode",
		// 		Parameters: map[string]interface{}{
		// 			"changesList": []map[string]interface{}{
		// 				{
		// 					"type":        "refactor",
		// 					"description": "Improve function structure",
		// 					"target":      "main.py",
		// 					"priority":    "medium",
		// 				},
		// 			},
		// 			"applyBlackFormatting": true,
		// 			"runLinter":            true,
		// 		},
		// 	},
		// 	Status:    types.TaskStatusPending,
		// 	CreatedAt: time.Now(),
		// },
		{
			ID:          "task4",
			Title:       "Test API Implementation",
			Description: "Run comprehensive tests for Python API",
			Requirements: types.TaskRequirement{
				SkillPath: types.TaskPath{"Development", "Testing", "Python"},
				Action:    "testCode",
				Parameters: map[string]interface{}{
					"testType":       "unit",
					"requirePassing": true,
					"testInstructions": []map[string]interface{}{
						{
							"description": "Test API endpoints",
							"assertions":  []string{"test_status_code", "test_response_format"},
							"testType":    "unit",
						},
					},
					"codeToTest":      "def example(): return True",
					"minimumCoverage": 80.0,
				},
			},
			Status:    types.TaskStatusPending,
			CreatedAt: time.Now(),
		},
	}

	// Add a small delay before sending tasks
	time.Sleep(1 * time.Second)

	// Route tasks
	for _, task := range tasks {
		log.Printf("Routing task: %s (Type: %s, Required Skills: %v)",
			task.Title, task.ID, task.Requirements.SkillPath)

		if err := router.RouteTask(ctx, task); err != nil {
			log.Printf("Failed to route task: %v", err)
			continue
		}

		// Add delay between tasks for readable logs
		time.Sleep(7 * time.Second)
	}

	// Wait for shutdown signal
	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("Shutting down...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Print metrics before shutdown
	log.Println("\nTask Execution Metrics:")
	log.Println("----------------------")

	// Shutdown agents
	for _, agent := range agents {
		if err := agent.Stop(shutdownCtx); err != nil {
			log.Printf("Error stopping agent %s: %v", agent.ID, err)
		}
	}
	// Close broker
	if err := broker.Close(); err != nil {
		log.Printf("Error closing broker: %v", err)
	}
}
