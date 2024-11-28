package engine

import (
    "context"
    "time"

    "github.com/Relax-N-Tax/AgentNexus/internal/task/types"
    "github.com/Relax-N-Tax/AgentNexus/pkg/circuit"
    "github.com/Relax-N-Tax/AgentNexus/pkg/metrics"

    "golang.org/x/time/rate"
)

// DefaultTaskExecutor Implementation
type DefaultTaskExecutor struct {
    registry       TaskRegistry
    circuitBreaker *CircuitBreaker
    rateLimiter    *rate.Limiter
    metrics        *ExecutorMetrics
}

// Metrics Collection
type ExecutorMetrics struct {
    taskLatencies    *metrics.Histogram
    taskSuccess      *metrics.Counter
    taskFailures     *metrics.Counter
    activeExecutions *metrics.Gauge
}

func NewExecutorMetrics() *ExecutorMetrics {
    return &ExecutorMetrics{
        taskLatencies:    metrics.NewHistogram(metrics.HistogramOpts{
            Name: "task_execution_duration_seconds",
            Help: "Task execution duration in seconds",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
        }),
        taskSuccess:      metrics.NewCounter(metrics.CounterOpts{
            Name: "task_executions_success_total",
            Help: "Total number of successful task executions",
        }),
        taskFailures:     metrics.NewCounter(metrics.CounterOpts{
            Name: "task_executions_failure_total",
            Help: "Total number of failed task executions",
        }),
        activeExecutions: metrics.NewGauge(metrics.GaugeOpts{
            Name: "task_executions_active",
            Help: "Number of currently active task executions",
        }),
    }
}

func NewDefaultTaskExecutor(registry TaskRegistry) *DefaultTaskExecutor {
    return &DefaultTaskExecutor{
        registry:       registry,
        circuitBreaker: NewCircuitBreaker(),
        rateLimiter:    rate.NewLimiter(rate.Limit(100), 200), // 100 rps, burst of 200
        metrics:        NewExecutorMetrics(),
    }
}
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func (e *DefaultTaskExecutor) ExecuteTask(ctx context.Context, task *TaskInstance) error {
    // Check circuit breaker
    if !e.circuitBreaker.AllowRequest() {
        return ErrCircuitOpen
    }

    // Apply rate limiting
    if err := e.rateLimiter.Wait(ctx); err != nil {
        return fmt.Errorf("rate limit exceeded: %v", err)
    }

    // Get task definition
    def, err := e.registry.GetDefinition(task.DefinitionID)
    if err != nil {
        return fmt.Errorf("failed to get task definition: %v", err)
    }

    // Apply retry policy with exponential backoff
    var lastErr error
    for attempt := 0; attempt < def.RetryPolicy.MaxAttempts; attempt++ {
        if attempt > 0 {
            backoff := def.RetryPolicy.BackoffInitial * time.Duration(math.Pow(def.RetryPolicy.BackoffFactor, float64(attempt)))
            if backoff > def.RetryPolicy.BackoffMax {
                backoff = def.RetryPolicy.BackoffMax
            }
            time.Sleep(backoff)
        }

        // Execute with timeout
        execCtx, cancel := context.WithTimeout(ctx, def.Timeout)
        err = e.executeWithMetrics(execCtx, task, def)
        cancel()

        if err == nil {
            e.circuitBreaker.RecordSuccess()
            return nil
        }

        lastErr = err
        e.circuitBreaker.RecordFailure()
        
        // Log retry attempt
        log.Printf("Task %s attempt %d failed: %v", task.ID, attempt+1, err)
    }

    return fmt.Errorf("all retry attempts failed: %v", lastErr)
}

func (e *DefaultTaskExecutor) executeWithMetrics(ctx context.Context, task *TaskInstance, def *AITaskDefinition) error {
    start := time.Now()
    err := e.executeTaskImpl(ctx, task, def)
    duration := time.Since(start)

    // Record metrics
    e.metrics.RecordExecution(def.TaskID, duration, err == nil)
    
    return err
}


func (e *DefaultTaskExecutor) executeSummarizationTask(ctx context.Context, task *TaskInstance, def *AITaskDefinition) error {
    input, ok := task.Input["prompt"].(string)
    if !ok {
        return fmt.Errorf("invalid input format")
    }

    // Simulate summarization processing
    time.Sleep(1 * time.Second)

    // Create result
    task.Result = &TaskResult{
        Output: map[string]interface{}{
            "response": fmt.Sprintf("Summary of: %s", input[:min(len(input), 50)]),
            "metadata": map[string]interface{}{
                "originalLength": len(input),
                "summaryLength": 50,
            },
        },
        Metrics: ExecutionMetrics{
            ProcessingTime: 1 * time.Second,
            InputSize:     len(input),
            OutputSize:    50,
        },
    }

    return nil
}

func (e *DefaultTaskExecutor) ValidateInput(def *AITaskDefinition, input map[string]interface{}) error {
    // Implementation of input validation against schema
    return nil
}

func (e *DefaultTaskExecutor) ValidateOutput(def *AITaskDefinition, output map[string]interface{}) error {
    // Implementation of output validation against schema
    return nil
}

// Enhanced Task Executor with retries and circuit breaker
type Health struct {
    Status    string    `json:"status"`
    LastCheck time.Time `json:"lastCheck"`
    Errors    []string  `json:"errors,omitempty"`
}

type TaskExecutor interface {
    ExecuteTask(ctx context.Context, task *TaskInstance) error
    ValidateInput(def *AITaskDefinition, input map[string]interface{}) error
    ValidateOutput(def *AITaskDefinition, output map[string]interface{}) error
    GetHealth() Health
}
