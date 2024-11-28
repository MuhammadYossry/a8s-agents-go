package main

import (
    "context"
    "log"
    "time"

    "github.com/MuhammadYossry/AgentNexus/internal/task"
    "github.com/MuhammadYossry/AgentNexus/internal/agent"
    "github.com/MuhammadYossry/AgentNexus/internal/broker"
    "github.com/MuhammadYossry/AgentNexus/internal/router"
    "github.com/MuhammadYossry/AgentNexus/pkg/metrics"
    "github.com/MuhammadYossry/AgentNexus/pkg/logging"
    "github.com/MuhammadYossry/AgentNexus/pkg/circuit"
)



// Example Usage
func Example() {
    // Initialize agent manager
    manager := NewAgentManager(ManagerConfig{
        HeartbeatInterval:   5 * time.Second,
        HealthCheckInterval: 30 * time.Second,
    })

    // Create and register agent
    agent := NewAgentImpl(AgentConfig{
        ID:          "agent-001",
        Name:        "NLP-Processor",
        Capabilities: []Capability{
            {
                Name:    "text-summarization",
                Version: "2.0",
                Properties: PropertySet{
                    MaxInputLength:  10000,
                    Language:       []string{"en", "es"},
                },
            },
        },
        QueueSize:    100,
        ReportingInterval: 10 * time.Second,
    })

    // Register agent
    if err := manager.RegisterAgent(agent.agent); err != nil {
        log.Fatalf("Failed to register agent: %v", err)
    }

    // Start agent
    if err := agent.Start(); err != nil {
        log.Fatalf("Failed to start agent: %v", err)
    }
}

func main() {
    // Initialize components
    registry := NewInMemoryTaskRegistry()
    store := NewInMemoryTaskStore()
    executor := NewDefaultTaskExecutor(registry)
    engine := NewTaskEngine(registry, executor, store)

    // Register summarization task definition
    def := &AITaskDefinition{
        TaskID:      "text-summarization-001",
        Name:        "Text Summarization",
        Description: "Generate a concise summary of a given text while preserving key information.",
        Version:     "1.1",
        InputSchema: SchemaDefinition{
            Type: "object",
            Properties: map[string]interface{}{
                "prompt": map[string]interface{}{
                    "type":        "string",
                    "description": "The text to be summarized",
                },
            },
            Required: []string{"prompt"},
        },
        Tags: []string{"nlp", "summarization"},
    }

    if err := registry.RegisterDefinition(def); err != nil {
        panic(err)
    }

    // Submit test task
    ctx := context.Background()
    input := map[string]interface{}{
        "prompt": "This is a sample text that needs to be summarized. It contains multiple sentences and ideas that should be condensed into a shorter version while maintaining the key points.",
    }

    task, err := engine.SubmitTask(ctx, def.TaskID, input)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Task submitted: %s\n", task.ID)

    // Wait for a moment to allow task processing
    time.Sleep(2 * time.Second)

    // Check task result
    completedTask, err := store.GetTask(task.ID)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Task status: %s\n", completedTask.Status.Phase)
    if completedTask.Result != nil {
        fmt.Printf("Summary: %s\n", completedTask.Result.Output["response"])
    }
}