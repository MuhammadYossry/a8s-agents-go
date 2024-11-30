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

	// Initialize components
	broker := NewPubSub()
	registry := NewCapabilityRegistry()
	router := NewTaskRouter(registry, broker)

	// Create agents with different capabilities
	agents := []*Agent{
		NewAgent("agent1", broker, NewTaskExecutor(), NewMetrics(), registry,
			[]string{"book_summary", "text_analysis", "book_task"}),
		NewAgent("agent2", broker, NewTaskExecutor(), NewMetrics(), registry,
			[]string{"video_generation", "image_processing", "video_task"}),
		NewAgent("agent3", broker, NewTaskExecutor(), NewMetrics(), registry,
			[]string{"market_analysis", "data_visualization"}),
		NewAgent("agent4", broker, NewTaskExecutor(), NewMetrics(), registry,
			[]string{"code_generation", "code_review"}),
		NewAgent("agent5", broker, NewTaskExecutor(), NewMetrics(), registry,
			[]string{"translation", "language_detection"}),
	}

	// Start agents
	for _, agent := range agents {
		if err := agent.Start(ctx); err != nil {
			log.Fatalf("Failed to start agent: %v", err)
		}
	}

	// Create tasks that match exactly one agent each
	tasks := []*Task{
		{
			ID:           "task1",
			Title:        "Summarize Harry Potter Book",
			Description:  "Create a summary of Harry Potter and the Philosopher's Stone",
			Type:         "book_task",
			Capabilities: []string{"book_summary"}, // matches agent1
			CreatedAt:    time.Now(),
		},
		{
			ID:           "task2",
			Title:        "Create Product Video",
			Description:  "Generate promotional video for new product",
			Type:         "video_task",
			Capabilities: []string{"video_generation"}, // matches agent2
			CreatedAt:    time.Now(),
		},
		{
			ID:           "task3",
			Title:        "Analyze EV Market",
			Description:  "Market analysis for EV industry",
			Type:         "analysis_task",
			Capabilities: []string{"market_analysis"}, // matches agent3
			CreatedAt:    time.Now(),
		},
		{
			ID:           "task4",
			Title:        "Review Python Codebase",
			Description:  "Code review for Python application",
			Type:         "code_task",
			Capabilities: []string{"code_review"}, // matches agent4
			CreatedAt:    time.Now(),
		},
		{
			ID:           "task5",
			Title:        "Translate Document",
			Description:  "Translate English to Spanish",
			Type:         "translation_task",
			Capabilities: []string{"translation"}, // matches agent5
			CreatedAt:    time.Now(),
		},
	}

	// Add a small delay before sending tasks to ensure all agents are ready
	time.Sleep(1 * time.Second)

	// Route tasks
	for _, task := range tasks {
		log.Printf("Enqueueing task: %s (requires capabilities: %v)", task.Title, task.Capabilities)
		if err := router.RouteTask(ctx, task); err != nil {
			log.Printf("Failed to route task: %v", err)
		}
		// Add a small delay between tasks to make logs more readable
		time.Sleep(500 * time.Millisecond)
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	_, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// XXX: Add agent shutdown logic via pubsub
	// if err := agent.Stop(shutdownCtx); err != nil {
	// 	log.Printf("Error during shutdown: %v", err)
	// }
}
