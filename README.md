![alt text](https://pbs.twimg.com/media/GZhrK4IWcAIb949?format=jpg&name=medium)

# AI Task Routing System

## Overview
A production-grade Go implementation of an AI task routing and execution system, designed for high availability, scalability, and reliability. The system efficiently manages and routes AI tasks to appropriate execution endpoints while ensuring robust error handling, monitoring, and performance optimization.

### Architecture
[Architecture](docs/architecture.md)

## Key Features

### Core Functionality
- Dynamic task routing and load balancing
- Comprehensive task lifecycle management
- Configurable retry policies with exponential backoff
- Circuit breaker pattern implementation
- Rate limiting and resource quotas

### Monitoring & Observability
- Structured logging with Zap logger
- Prometheus-compatible metrics
- Health check endpoints
- Error tracking and analysis
- Performance monitoring

### Security & Validation
- Content safety validation
- Input/output schema validation
- Rate limiting per client/task
- Resource quotas enforcement

### Reliability
- Circuit breaker pattern for fault tolerance
- Configurable retry policies
- Timeout handling
- Graceful degradation
- Error recovery mechanisms

## System Components

### Task Definition
```go
type AITaskDefinition struct {
    TaskID          string
    Name            string
    Version         string
    InputSchema     SchemaDefinition
    OutputSchema    SchemaDefinition
    Timeout         time.Duration
    RetryPolicy     RetryPolicy
    RateLimit       *RateLimit
    ResourceQuota   *ResourceQuota
}
```

### Task Executor
```go
type TaskExecutor interface {
    ExecuteTask(ctx context.Context, task *TaskInstance) error
    ValidateInput(def *AITaskDefinition, input map[string]interface{}) error
    ValidateOutput(def *AITaskDefinition, output map[string]interface{}) error
    GetHealth() Health
}
```

## Configuration

### Retry Policy
```go
type RetryPolicy struct {
    MaxAttempts     int
    BackoffInitial  time.Duration
    BackoffMax      time.Duration
    BackoffFactor   float64
}
```

### Rate Limiting
```go
type RateLimit struct {
    RequestsPerSecond int
    BurstSize        int
}
```

### Resource Quotas
```go
type ResourceQuota struct {
    MaxConcurrent    int
    MemoryLimit      string
    CPULimit         string
}
```

## Usage Examples

### Registering a Task Definition
```go
def := &AITaskDefinition{
    TaskID:      "text-summarization-001",
    Name:        "Text Summarization",
    Version:     "1.1",
    Timeout:     30 * time.Second,
    RetryPolicy: RetryPolicy{
        MaxAttempts:    3,
        BackoffInitial: time.Second,
        BackoffMax:     10 * time.Second,
        BackoffFactor:  2.0,
    },
    RateLimit: &RateLimit{
        RequestsPerSecond: 100,
        BurstSize:        200,
    },
}

if err := registry.RegisterDefinition(def); err != nil {
    log.Fatalf("Failed to register task: %v", err)
}
```

### Submitting a Task
```go
input := map[string]interface{}{
    "prompt": "Text to be summarized...",
}

task, err := engine.SubmitTask(ctx, "text-summarization-001", input)
if err != nil {
    log.Fatalf("Failed to submit task: %v", err)
}
```

## Monitoring

### Metrics
The system exposes the following Prometheus metrics:
- `task_execution_duration_seconds`: Histogram of task execution times
- `task_executions_success_total`: Counter of successful executions
- `task_executions_failure_total`: Counter of failed executions
- `task_executions_active`: Gauge of currently running tasks

### Health Checks
Health check endpoint returns:
- Circuit breaker status
- Task executor health
- System resource usage
- Recent error rates

## Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| EXECUTOR_MAX_CONCURRENT | Maximum concurrent tasks | 100 |
| CIRCUIT_BREAKER_THRESHOLD | Failures before opening | 5 |
| CIRCUIT_BREAKER_RESET_TIMEOUT | Time before retry | 30s |
| DEFAULT_TASK_TIMEOUT | Default task timeout | 30s |
| MAX_RETRY_ATTEMPTS | Maximum retry attempts | 3 |

## Production Considerations

### Scaling
- Horizontal scaling supported through consistent hashing
- Resource quotas prevent system overload
- Rate limiting per client/task type

### Reliability
- Circuit breaker prevents cascade failures
- Retry policies handle transient errors
- Timeout enforcement prevents resource exhaustion

### Monitoring
- Prometheus metrics for alerting
- Structured logging for debugging
- Health checks for load balancers

### Security
- Content validation prevents abuse
- Rate limiting prevents DoS
- Resource quotas ensure fair usage

## Error Handling

The system implements comprehensive error handling:
- Retries for transient failures
- Circuit breaking for systemic issues
- Detailed error logging
- Error categorization and tracking

## Best Practices

1. Configure appropriate timeouts for each task type
2. Set realistic rate limits based on resource capacity
3. Monitor error rates and latency metrics
4. Implement gradual rollouts of new task definitions
5. Regular health check monitoring

## To-Do List
- [ ] Add AI agent as executors for the task
- [ ] Add AI agent Workflow for agent collobration
- [ ] Implement distributed tracing
- [ ] Add support for multiple storage backends
- [ ] Enhance metrics collection
- [ ] Add support for task priorities
- [ ] Implement task result caching
