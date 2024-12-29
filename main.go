package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/orchestrator"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := orchestrator.Config{
		AgentsConfigPath:   "examples/agents_generated.json",
		AgentsConfigMDPath: "examples/readable_agents.md",
		InternalConfig: types.InternalAgentConfig{
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
		},
	}

	orchestrator, err := orchestrator.New(config)
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Start orchestrator
	if err := orchestrator.Start(ctx); err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}

	ctx, err = orchestrator.ProcessQuery(ctx,
		"I want to buid a REST API using Python  Fastapi Django Rest",
	)
	if err != nil {
		log.Fatal(err)
	}

	result := ctx.Value(types.TaskExtractionResultKey).(string)
	log.Printf("Extracted Task: %s\n", result)

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
		{
			ID:          "task3",
			Title:       "Improve Code Quality",
			Description: "Refactor and optimize Python codebase",
			Requirements: types.TaskRequirement{
				SkillPath: types.TaskPath{"Development", "Backend", "Python", "CodeGeneration"},
				Action:    "improveCode",
				Parameters: map[string]interface{}{
					"changesList": []map[string]interface{}{
						{
							"type":        "refactor",
							"description": "Improve function structure",
							"target":      "main.py",
							"priority":    "medium",
						},
					},
					"applyBlackFormatting": true,
					"runLinter":            true,
				},
			},
			Status:    types.TaskStatusPending,
			CreatedAt: time.Now(),
		},
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

	// Execute tasks
	if err := orchestrator.ExecuteTasks(ctx, tasks); err != nil {
		log.Printf("Error executing tasks: %v", err)
	}

	// Handle shutdown gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("Shutting down...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := orchestrator.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
