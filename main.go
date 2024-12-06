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
)

func LoadAgentConfig(filepath string, broker Broker, metrics *Metrics, registry *CapabilityRegistry) ([]*Agent, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	agents := make([]*Agent, 0, len(config.Agents))
	for _, def := range config.Agents {
		// ctx, cancel := context.WithCancel(context.Background())
		// agentID =
		taskExecutor := NewTaskExecutor(&def)
		agent := NewAgent(AgentID(def.ID), broker, taskExecutor, metrics,
			registry, &def)
		// Type:         def.Type,
		// Description:  def.Description,
		// BaseURL:      def.BaseURL,
		// TaskTypes:    def.TaskTypes,
		// SkillsByType: def.SkillsByType,
		// Actions:      def.Actions,
		// broker:       ,
		// executor:     executor,
		// metrics:      metrics,
		// registry:     registry,

		agents = append(agents, agent)
	}

	return agents, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize components
	broker := NewPubSub()
	metrics := NewMetrics()
	registry := NewCapabilityRegistry()
	router := NewTaskRouter(registry, broker, metrics)

	agents, err := LoadAgentConfig("examples/agents.json", broker,
		metrics, registry)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	for _, agent := range agents {
		fmt.Printf("Agent ID: %s\n", agent.ID)
		fmt.Printf("Task Types: %v\n", agent.TaskTypes)

		for taskType, skills := range agent.SkillsByType {
			fmt.Printf("Skills for %s: %v\n", taskType, skills)
		}
	}

	// wf_agents := []*Agent{
	// 	NewAgent(
	// 		"wf_agent1",
	// 		broker,
	// 		NewTaskExecutor(),
	// 		NewMetrics(),
	// 		registry,
	// 		[]string{"data_ingest_task"},
	// 		map[string][]string{
	// 			"data_ingest_task": {"data_ingestion", "data_validation"},
	// 		},
	// 	),
	// 	NewAgent(
	// 		"wf_agent2",
	// 		broker,
	// 		NewTaskExecutor(),
	// 		NewMetrics(),
	// 		registry,
	// 		[]string{"data_transform_task"},
	// 		map[string][]string{
	// 			"data_transform_task": {"data_transformation", "data_cleaning"},
	// 		},
	// 	),
	// 	NewAgent(
	// 		"wf_agent3",
	// 		broker,
	// 		NewTaskExecutor(),
	// 		NewMetrics(),
	// 		registry,
	// 		[]string{"data_validation_task"},
	// 		map[string][]string{
	// 			"data_validation_task": {"data_validation", "quality_check"},
	// 		},
	// 	),
	// 	NewAgent(
	// 		"wf_agent4",
	// 		broker,
	// 		NewTaskExecutor(),
	// 		NewMetrics(),
	// 		registry,
	// 		[]string{"pipeline_monitor_task"},
	// 		map[string][]string{
	// 			"pipeline_monitor_task": {"pipeline_monitoring", "status_tracking"},
	// 		},
	// 	),
	// }

	// Start agents
	for _, agent := range agents {
		if err := agent.Start(ctx); err != nil {
			log.Fatalf("Failed to start agent: %v", err)
		}
	}

	// Todo: refactor to remove Convert []*Agent to []Agent when passing to NewWorkflow
	// wfAgentsSlice := make([]Agent, len(wf_agents))
	// for i, agent := range wf_agents {
	// 	wfAgentsSlice[i] = *agent
	// }

	// Create and start the workflow with its dedicated agents
	// dataPipelineWorkflow := NewWorkflow(
	// 	"data_workflow_1",
	// 	broker,
	// 	metrics,
	// 	registry,
	// 	wf_agents, // Pass workflow-specific agents
	// 	[]string{"data_pipeline_task"},
	// 	map[string][]string{
	// 		"data_pipeline_task": {
	// 			"data_ingestion",
	// 			"data_transformation",
	// 			"data_validation",
	// 			"pipeline_monitoring",
	// 		},
	// 	},
	// )

	// if err := dataPipelineWorkflow.Start(ctx); err != nil {
	// 	log.Fatalf("Failed to start workflow: %v", err)
	// }

	// Create sample tasks
	generatTaskData := `{
        "code_requirements": {
            "description": "Create a REST API endpoint",
            "required_functions": ["get_user", "create_user"],
            "dependencies": ["fastapi", "sqlalchemy"],
            "python_version": "3.9",
            "testing_requirements": ["pytest"]
        },
        "style_guide": "PEP8",
        "include_tests": true,
        "documentation_level": "detailed"
    }`

	tasks := []*Task{
		{
			ID:             "task1",
			Title:          "Create a REST API endpoint",
			Description:    "Create a REST API endpoint using Pythoon",
			Type:           "pythonCodeTask",
			SkillsRequired: []string{"generateCode", "generateTests"},
			Payload:        []byte(generatTaskData),
			Status:         "Pending",
			RetryCount:     0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Time{},
		},
		// {
		// 	ID:             "task2",
		// 	Title:          "Create Product Video",
		// 	Description:    "Generate promotional video for new product",
		// 	Type:           "video_task",
		// 	SkillsRequired: []string{"video_generation", "video_editing"},
		// 	CreatedAt:      time.Now(),
		// },
		// {
		// 	ID:             "task3",
		// 	Title:          "Market Analysis Report",
		// 	Description:    "Analyze EV market trends with visualizations",
		// 	Type:           "analysis_task",
		// 	SkillsRequired: []string{"market_analysis", "data_visualization"},
		// 	CreatedAt:      time.Now(),
		// },
		// {
		// 	ID:             "task4",
		// 	Title:          "Optimize Python Codebase",
		// 	Description:    "Review and optimize Python application performance",
		// 	Type:           "code_task",
		// 	SkillsRequired: []string{"code_review", "code_optimization"},
		// 	CreatedAt:      time.Now(),
		// },
		// {
		// 	ID:             "task5",
		// 	Title:          "Translate and Proofread Document",
		// 	Description:    "Translate English document to Spanish and proofread",
		// 	Type:           "translatio	n_task",
		// 	SkillsRequired: []string{"translation", "proofreading"},
		// 	CreatedAt:      time.Now(),
		// },
		// {
		// 	ID:          "task6",
		// 	Title:       "Process Customer Data Pipeline",
		// 	Description: "Execute end-to-end data processing pipeline for customer data(Workflow specfic)",
		// 	Type:        "data_pipeline_task",
		// 	SkillsRequired: []string{
		// 		"data_ingestion",
		// 		"data_transformation",
		// 		"data_validation",
		// 		"pipeline_monitoring",
		// 	},
		// 	CreatedAt: time.Now(),
		// },
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
	// taskTypes := []string{
	// 	"book_task",
	// 	"video_task",
	// 	"analysis_task",
	// 	"code_task",
	// 	"translation_task",
	// 	"language_task",
	// 	"image_task",
	// 	"data_pipeline_task",
	// 	"data_ingest_task",
	// 	"data_transform_task",
	// 	"data_validation_task",
	// 	"pipeline_monitor_task",
	// }

	// allMetrics := metrics.GetAllMetrics()
	// for _, taskType := range taskTypes {
	// 	if m, exists := allMetrics[taskType]; exists {
	// 		log.Printf("%s:\n  - Completed: %d\n  - Failed: %d\n  - Routing Successes: %d\n  - Routing Failures: %d",
	// 			taskType,
	// 			m.TasksCompleted,
	// 			m.TasksFailed,
	// 			m.RoutingSuccesses,
	// 			m.RoutingFailures)
	// 	} else {
	// 		log.Printf("%s: No metrics recorded", taskType)
	// 	}
	// }

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
