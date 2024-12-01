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
	metrics := NewMetrics()
	registry := NewCapabilityRegistry()
	router := NewTaskRouter(registry, broker, metrics)

	// Create agents with different capabilities
	agents := []*Agent{
		NewAgent(
			"agent1",
			broker,
			NewTaskExecutor(),
			NewMetrics(),
			registry,
			[]string{"book_task"},
			map[string][]string{
				"book_task": {"book_summary", "text_analysis", "content_review"},
			},
		),
		NewAgent(
			"agent2",
			broker,
			NewTaskExecutor(),
			NewMetrics(),
			registry,
			[]string{"video_task", "image_task"},
			map[string][]string{
				"video_task": {"video_generation", "video_editing"},
				"image_task": {"image_processing", "image_analysis"},
			},
		),
		NewAgent(
			"agent3",
			broker,
			NewTaskExecutor(),
			NewMetrics(),
			registry,
			[]string{"analysis_task"},
			map[string][]string{
				"analysis_task": {"market_analysis", "data_visualization", "trend_analysis"},
			},
		),
		NewAgent(
			"agent4",
			broker,
			NewTaskExecutor(),
			NewMetrics(),
			registry,
			[]string{"code_task"},
			map[string][]string{
				"code_task": {"code_generation", "code_review", "code_optimization"},
			},
		),
		NewAgent(
			"agent5",
			broker,
			NewTaskExecutor(),
			NewMetrics(),
			registry,
			[]string{"translation_task", "language_task"},
			map[string][]string{
				"translation_task": {"translation", "proofreading"},
				"language_task":    {"language_detection", "sentiment_analysis"},
			},
		),
	}

	wf_agents := []*Agent{
		NewAgent(
			"wf_agent1",
			broker,
			NewTaskExecutor(),
			NewMetrics(),
			registry,
			[]string{"data_ingest_task"},
			map[string][]string{
				"data_ingest_task": {"data_ingestion", "data_validation"},
			},
		),
		NewAgent(
			"wf_agent2",
			broker,
			NewTaskExecutor(),
			NewMetrics(),
			registry,
			[]string{"data_transform_task"},
			map[string][]string{
				"data_transform_task": {"data_transformation", "data_cleaning"},
			},
		),
		NewAgent(
			"wf_agent3",
			broker,
			NewTaskExecutor(),
			NewMetrics(),
			registry,
			[]string{"data_validation_task"},
			map[string][]string{
				"data_validation_task": {"data_validation", "quality_check"},
			},
		),
		NewAgent(
			"wf_agent4",
			broker,
			NewTaskExecutor(),
			NewMetrics(),
			registry,
			[]string{"pipeline_monitor_task"},
			map[string][]string{
				"pipeline_monitor_task": {"pipeline_monitoring", "status_tracking"},
			},
		),
	}

	// Start agents
	for _, agent := range agents {
		if err := agent.Start(ctx); err != nil {
			log.Fatalf("Failed to start agent: %v", err)
		}
	}

	// Todo: refactor to remove Convert []*Agent to []Agent when passing to NewWorkflow
	wfAgentsSlice := make([]Agent, len(wf_agents))
	for i, agent := range wf_agents {
		wfAgentsSlice[i] = *agent
	}

	// Create and start the workflow with its dedicated agents
	dataPipelineWorkflow := NewWorkflow(
		"data_workflow_1",
		broker,
		metrics,
		registry,
		wf_agents, // Pass workflow-specific agents
		[]string{"data_pipeline_task"},
		map[string][]string{
			"data_pipeline_task": {
				"data_ingestion",
				"data_transformation",
				"data_validation",
				"pipeline_monitoring",
			},
		},
	)

	if err := dataPipelineWorkflow.Start(ctx); err != nil {
		log.Fatalf("Failed to start workflow: %v", err)
	}

	// Create sample tasks
	tasks := []*Task{
		{
			ID:             "task1",
			Title:          "Summarize Harry Potter Book",
			Description:    "Create a summary of Harry Potter and the Philosopher's Stone",
			Type:           "book_task",
			SkillsRequired: []string{"book_summary", "content_review"},
			CreatedAt:      time.Now(),
		},
		{
			ID:             "task2",
			Title:          "Create Product Video",
			Description:    "Generate promotional video for new product",
			Type:           "video_task",
			SkillsRequired: []string{"video_generation", "video_editing"},
			CreatedAt:      time.Now(),
		},
		{
			ID:             "task3",
			Title:          "Market Analysis Report",
			Description:    "Analyze EV market trends with visualizations",
			Type:           "analysis_task",
			SkillsRequired: []string{"market_analysis", "data_visualization"},
			CreatedAt:      time.Now(),
		},
		{
			ID:             "task4",
			Title:          "Optimize Python Codebase",
			Description:    "Review and optimize Python application performance",
			Type:           "code_task",
			SkillsRequired: []string{"code_review", "code_optimization"},
			CreatedAt:      time.Now(),
		},
		{
			ID:             "task5",
			Title:          "Translate and Proofread Document",
			Description:    "Translate English document to Spanish and proofread",
			Type:           "translatio	n_task",
			SkillsRequired: []string{"translation", "proofreading"},
			CreatedAt:      time.Now(),
		},
		{
			ID:          "task6",
			Title:       "Process Customer Data Pipeline",
			Description: "Execute end-to-end data processing pipeline for customer data(Workflow specfic)",
			Type:        "data_pipeline_task",
			SkillsRequired: []string{
				"data_ingestion",
				"data_transformation",
				"data_validation",
				"pipeline_monitoring",
			},
			CreatedAt: time.Now(),
		},
	}

	// Add a small delay before sending tasks
	time.Sleep(1 * time.Second)

	// Route tasks
	for _, task := range tasks {
		log.Printf("Routing task: %s (Type: %s, Required Skills: %v)",
			task.Title, task.Type, task.SkillsRequired)

		if err := router.RouteTask(ctx, task); err != nil {
			log.Printf("Failed to route task: %v", err)
			continue
		}

		// Add delay between tasks for readable logs
		time.Sleep(500 * time.Millisecond)
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Print metrics before shutdown
	log.Println("\nTask Execution Metrics:")
	log.Println("----------------------")

	// Define all task types we want to report on
	taskTypes := []string{
		"book_task",
		"video_task",
		"analysis_task",
		"code_task",
		"translation_task",
		"language_task",
		"image_task",
		"data_pipeline_task",
		"data_ingest_task",
		"data_transform_task",
		"data_validation_task",
		"pipeline_monitor_task",
	}

	allMetrics := metrics.GetAllMetrics()
	for _, taskType := range taskTypes {
		if m, exists := allMetrics[taskType]; exists {
			log.Printf("%s:\n  - Completed: %d\n  - Failed: %d\n  - Routing Successes: %d\n  - Routing Failures: %d",
				taskType,
				m.TasksCompleted,
				m.TasksFailed,
				m.RoutingSuccesses,
				m.RoutingFailures)
		} else {
			log.Printf("%s: No metrics recorded", taskType)
		}
	}

	// Shutdown agents
	for _, agent := range agents {
		if err := agent.Stop(shutdownCtx); err != nil {
			log.Printf("Error stopping agent: %v", err)
		}
	}

	// Close broker
	if err := broker.Close(); err != nil {
		log.Printf("Error closing broker: %v", err)
	}
}
