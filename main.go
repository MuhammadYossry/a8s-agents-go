package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/core"
	dapr "github.com/dapr/go-sdk/client"
)

// Payload represents the inner payload structure
type Payload struct {
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
}

// RequestEvent represents the structure of events we'll publish
type RequestEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   Payload   `json:"payload"`
}

func publishEvent(ctx context.Context, client dapr.Client, eventID string) error {
	pubsubName := "internal-agents"
	topic := "request-generator"

	event := RequestEvent{
		ID:        eventID,
		Type:      "code_generation",
		Timestamp: time.Now().UTC(),
		Payload: Payload{
			Action: "generate",
			Parameters: map[string]interface{}{
				"language": "python",
				"task":     "implement basic calculator",
			},
		},
	}

	// Convert event to JSON and log it
	eventData, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return err
	}

	log.Printf("Publishing event to pubsub '%s' on topic '%s':\n%s",
		pubsubName, topic, string(eventData))

	// Publish the event
	err = client.PublishEvent(ctx, pubsubName, topic, eventData)
	if err != nil {
		return err
	}

	log.Printf("Successfully published event: %s to topic: %s", event.ID, topic)
	return nil
}

func startEventPublisher(ctx context.Context) error {
	// Create the Dapr client
	client, err := dapr.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Publish initial event
	if err := publishEvent(ctx, client, "evt-001"); err != nil {
		return err
	}

	// Start a timer to publish events periodically
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	eventCounter := 2
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			eventID := fmt.Sprintf("evt-%03d", eventCounter)
			if err := publishEvent(ctx, client, eventID); err != nil {
				log.Printf("Failed to publish event %s: %v", eventID, err)
			}
			eventCounter++
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker := core.NewPubSub()
	metrics := core.NewMetrics()
	registry := core.NewCapabilityRegistry()
	router := core.NewTaskRouter(registry, broker, metrics)

	loader := core.NewAgentLoader(broker, metrics, registry)
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

	tasks := []*core.Task{
		{
			ID:          "task1",
			Title:       "Create a REST API endpoint",
			Description: "Create a REST API endpoint using Python",
			Requirements: core.TaskRequirement{
				SkillPath: core.TaskPath{"Development", "Backend", "Python", "CodeGeneration"},
				Action:    "generateCode",
				Parameters: map[string]interface{}{
					"framework": "FastAPI",
				},
			},
			Payload:   []byte(generatTaskData),
			Status:    core.TaskStatusPending,
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

	// Start the event publisher in a separate goroutine
	go func() {
		if err := startEventPublisher(ctx); err != nil {
			log.Printf("Event publisher error: %v", err)
		}
	}()

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
