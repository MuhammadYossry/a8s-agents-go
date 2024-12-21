package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker := NewPubSub()
	metrics := NewMetrics()
	registry := NewCapabilityRegistry()
	router := NewTaskRouter(registry, broker, metrics)

	loader := NewAgentLoader(broker, metrics, registry)
	agents, err := loader.LoadAgents("examples/agents_generated.json")
	if err != nil {
		log.Fatalf("Failed to load agents: %v", err)
	}

	// Start agents
	for _, agent := range agents {
		if err := agent.Start(ctx); err != nil {
			log.Fatalf("Failed to start agent %s: %v", agent.ID, err)
		}
	}

	generatTaskData := `{
  "codeRequirements": {
    "language": "Python",
    "framework": "FastAPI",
    "description": "Create a web application for managing TODO items with basic CRUD operations",
    "requirements": [
      "FastAPI",
      "Pydantic",
      "CRUD functionality"
    ],
    "requiredFunctions": [
      "create_todo",
      "read_todo",
      "update_todo",
      "delete_todo"
    ],
    "testingRequirements": [
      "test_todo_creation",
      "test_todo_retrieval",
      "test_todo_update",
      "test_todo_deletion"
    ],
    "codingStyle": {
      "patterns": [
        "REST API",
        "Clean Architecture"
      ],
      "conventions": [
        "PEP 8",
        "FastAPI best practices"
      ]
    }
  },
  "styleGuide": {
    "formatting": "black",
    "maxLineLength": 88
  },
  "includeTests": true,
  "documentationLevel": "detailed"
}
`

	tasks := []*Task{
		{
			ID:          "task1",
			Title:       "Create a REST API endpoint",
			Description: "Create a REST API endpoint using Python",
			Requirements: TaskRequirement{
				SkillPath: TaskPath{"Development", "Backend", "Python", "CodeGeneration"},
				Action:    "generateCode",
				Parameters: map[string]interface{}{
					"framework": "FastAPI",
				},
			},
			Payload:   []byte(generatTaskData),
			Status:    TaskStatusPending,
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
		time.Sleep(500 * time.Millisecond)
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
