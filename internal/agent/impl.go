package agent

import (
    "time"

    "github.com/Relax-N-Tax/AgentNexus/internal/agent/types"
    "github.com/Relax-N-Tax/AgentNexus/internal/broker"
    "github.com/Relax-N-Tax/AgentNexus/internal/task/types"
    "github.com/Relax-N-Tax/AgentNexus/pkg/metrics"
)

// Agent Implementation
type AgentImpl struct {
    agent          *AIAgent
    taskQueue      *TaskQueue
    executor       *TaskExecutor
    reporter       *StatusReporter
    metrics        *MetricsCollector
}

func NewAgentImpl(config AgentConfig) *AgentImpl {
    return &AgentImpl{
        agent:     NewAIAgent(config),
        taskQueue: NewTaskQueue(config.QueueSize),
        executor:  NewTaskExecutor(config.ExecutorConfig),
        reporter:  NewStatusReporter(config.ReportingInterval),
        metrics:   NewMetricsCollector(config.MetricsConfig),
    }
}

func (a *AgentImpl) Start() error {
    // Initialize components
    if err := a.initialize(); err != nil {
        return err
    }

    // Start processing tasks
    go a.processTaskQueue()

    // Start reporting status
    go a.reportStatus()

    // Start collecting metrics
    go a.collectMetrics()

    return nil
}

// Task Processing
func (a *AgentImpl) processTaskQueue() {
    for task := range a.taskQueue.Tasks {
        // Update agent status
        a.agent.Status.State = "busy"
        
        // Process task
        start := time.Now()
        result, err := a.executor.ExecuteTask(task)
        duration := time.Since(start)

        // Update metrics
        a.metrics.RecordTaskExecution(task, duration, err)

        // Report result
        a.reporter.ReportTaskCompletion(task, result, err)

        // Update agent status
        a.agent.Status.State = "online"
        a.agent.Status.CurrentLoad = float64(a.taskQueue.Size()) / float64(a.taskQueue.Capacity())
    }
}