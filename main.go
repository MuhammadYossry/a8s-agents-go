package main

import (
    "context"
    "fmt"
    "sync"
    "time"
)

// Core Types Improvements
type AITaskDefinition struct {
    TaskID          string            `json:"taskId"`
    Name            string            `json:"name"`
    Description     string            `json:"description"`
    Version         string            `json:"version"`
    InputSchema     SchemaDefinition  `json:"inputSchema"`
    OutputSchema    SchemaDefinition  `json:"outputSchema"`
    Capabilities    []Capability      `json:"capabilities"`
    Constraints     TaskConstraints   `json:"constraints"`
    Performance     PerformanceSpec   `json:"performance"`
    Examples        []TaskExample     `json:"examples"`
    Tags            []string          `json:"tags"`
    // Added fields for better production readiness
    Owner           string            `json:"owner"`
    Status          string            `json:"status"`  // active, deprecated, beta
    Timeout         time.Duration     `json:"timeout"`
    RetryPolicy     RetryPolicy      `json:"retryPolicy"`
    RateLimit       *RateLimit       `json:"rateLimit,omitempty"`
    ResourceQuota   *ResourceQuota   `json:"resourceQuota,omitempty"`
}

type RetryPolicy struct {
    MaxAttempts     int           `json:"maxAttempts"`
    BackoffInitial  time.Duration `json:"backoffInitial"`
    BackoffMax      time.Duration `json:"backoffMax"`
    BackoffFactor   float64       `json:"backoffFactor"`
}

type RateLimit struct {
    RequestsPerSecond int     `json:"requestsPerSecond"`
    BurstSize        int     `json:"burstSize"`
}

type ResourceQuota struct {
    MaxConcurrent    int    `json:"maxConcurrent"`
    MemoryLimit      string `json:"memoryLimit"`  // e.g., "512Mi"
    CPULimit         string `json:"cpuLimit"`     // e.g., "0.5"
}

type SchemaDefinition struct {
    Type       string                 `json:"type"`
    Properties map[string]interface{} `json:"properties"`
    Required   []string              `json:"required"`
}

type Capability struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}

type TaskConstraints struct {
    MaxInputLength     int      `json:"maxInputLength"`
    MaxOutputLength    int      `json:"maxOutputLength"`
    SupportedLanguages []string `json:"supportedLanguages"`
    ContentFilters    []string `json:"contentFilters"`
}

type PerformanceSpec struct {
    AverageLatency string `json:"averageLatency"`
    Throughput     string `json:"throughput"`
}

type TaskExample struct {
    Input  map[string]interface{} `json:"input"`
    Output map[string]interface{} `json:"output"`
}

// Task Runtime Types
type TaskInstance struct {
    ID            string                 `json:"id"`
    DefinitionID  string                 `json:"definitionId"`
    Input         map[string]interface{} `json:"input"`
    Status        TaskStatus             `json:"status"`
    CreatedAt     time.Time             `json:"createdAt"`
    StartedAt     *time.Time            `json:"startedAt,omitempty"`
    CompletedAt   *time.Time            `json:"completedAt,omitempty"`
    Result        *TaskResult            `json:"result,omitempty"`
}

type TaskStatus struct {
    Phase     string `json:"phase"` // pending, running, completed, failed
    Message   string `json:"message,omitempty"`
    Progress  int    `json:"progress"`
    AttemptID string `json:"attemptId,omitempty"`
}

type TaskResult struct {
    Output   map[string]interface{} `json:"output"`
    Metrics  ExecutionMetrics      `json:"metrics"`
    Error    string                `json:"error,omitempty"`
}

type ExecutionMetrics struct {
    ProcessingTime time.Duration `json:"processingTime"`
    InputSize      int          `json:"inputSize"`
    OutputSize     int          `json:"outputSize"`
}

// Registry Interface
type TaskRegistry interface {
    RegisterDefinition(def *AITaskDefinition) error
    GetDefinition(id string) (*AITaskDefinition, error)
    ListDefinitions(tags []string) ([]*AITaskDefinition, error)
    UpdateDefinition(def *AITaskDefinition) error
}

// In-Memory Registry Implementation
type InMemoryTaskRegistry struct {
    mu           sync.RWMutex
    definitions  map[string]*AITaskDefinition
}

func NewInMemoryTaskRegistry() *InMemoryTaskRegistry {
    return &InMemoryTaskRegistry{
        definitions: make(map[string]*AITaskDefinition),
    }
}

func (r *InMemoryTaskRegistry) RegisterDefinition(def *AITaskDefinition) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.definitions[def.TaskID]; exists {
        return fmt.Errorf("task definition %s already exists", def.TaskID)
    }

    r.definitions[def.TaskID] = def
    return nil
}

func (r *InMemoryTaskRegistry) GetDefinition(id string) (*AITaskDefinition, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    def, exists := r.definitions[id]
    if !exists {
        return nil, fmt.Errorf("task definition %s not found", id)
    }
    return def, nil
}

func (r *InMemoryTaskRegistry) ListDefinitions(tags []string) ([]*AITaskDefinition, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    if len(tags) == 0 {
        defs := make([]*AITaskDefinition, 0, len(r.definitions))
        for _, def := range r.definitions {
            defs = append(defs, def)
        }
        return defs, nil
    }

    tagSet := make(map[string]bool)
    for _, tag := range tags {
        tagSet[tag] = true
    }

    var defs []*AITaskDefinition
    for _, def := range r.definitions {
        for _, tag := range def.Tags {
            if tagSet[tag] {
                defs = append(defs, def)
                break
            }
        }
    }
    return defs, nil
}

func (r *InMemoryTaskRegistry) UpdateDefinition(def *AITaskDefinition) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.definitions[def.TaskID]; !exists {
        return fmt.Errorf("task definition %s not found", def.TaskID)
    }

    r.definitions[def.TaskID] = def
    return nil
}

// Enhanced Task Executor with retries and circuit breaker
type TaskExecutor interface {
    ExecuteTask(ctx context.Context, task *TaskInstance) error
    ValidateInput(def *AITaskDefinition, input map[string]interface{}) error
    ValidateOutput(def *AITaskDefinition, output map[string]interface{}) error
    GetHealth() Health
}

type Health struct {
    Status    string    `json:"status"`
    LastCheck time.Time `json:"lastCheck"`
    Errors    []string  `json:"errors,omitempty"`
}


type InMemoryTaskStore struct {
    mu    sync.RWMutex
    tasks map[string]*TaskInstance
}

func NewInMemoryTaskStore() *InMemoryTaskStore {
    return &InMemoryTaskStore{
        tasks: make(map[string]*TaskInstance),
    }
}

func (s *InMemoryTaskStore) SaveTask(task *TaskInstance) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.tasks[task.ID]; exists {
        return fmt.Errorf("task %s already exists", task.ID)
    }

    s.tasks[task.ID] = task
    return nil
}

func (s *InMemoryTaskStore) GetTask(id string) (*TaskInstance, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    task, exists := s.tasks[id]
    if !exists {
        return nil, fmt.Errorf("task %s not found", id)
    }
    return task, nil
}

func (s *InMemoryTaskStore) UpdateTask(task *TaskInstance) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.tasks[task.ID]; !exists {
        return fmt.Errorf("task %s not found", task.ID)
    }

    s.tasks[task.ID] = task
    return nil
}

func (s *InMemoryTaskStore) ListTasks(filter TaskFilter) ([]*TaskInstance, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var tasks []*TaskInstance
    for _, task := range s.tasks {
        if matchesFilter(task, filter) {
            tasks = append(tasks, task)
        }
    }
    return tasks, nil
}

// DefaultTaskExecutor Implementation
type DefaultTaskExecutor struct {
    registry       TaskRegistry
    circuitBreaker *CircuitBreaker
    rateLimiter    *rate.Limiter
    metrics        *ExecutorMetrics
}

func NewDefaultTaskExecutor(registry TaskRegistry) *DefaultTaskExecutor {
    return &DefaultTaskExecutor{
        registry:       registry,
        circuitBreaker: NewCircuitBreaker(),
        rateLimiter:    rate.NewLimiter(rate.Limit(100), 200), // 100 rps, burst of 200
        metrics:        NewExecutorMetrics(),
    }
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

// Helper functions
func matchesFilter(task *TaskInstance, filter TaskFilter) bool {
    if filter.DefinitionID != "" && task.DefinitionID != filter.DefinitionID {
        return false
    }
    if filter.Status != "" && task.Status.Phase != filter.Status {
        return false
    }
    if filter.TimeRange != nil {
        if task.CreatedAt.Before(filter.TimeRange.Start) || task.CreatedAt.After(filter.TimeRange.End) {
            return false
        }
    }
    return true
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}


// Task Engine
type TaskEngine struct {
    registry    TaskRegistry
    executor    TaskExecutor
    store       TaskStore
    validator   *TaskValidator
    metrics     *EngineMetrics
    errorBuffer *ring.Buffer
}

func NewTaskEngine(registry TaskRegistry, executor TaskExecutor, store TaskStore) *TaskEngine {
    return &TaskEngine{
        registry:    registry,
        executor:    executor,
        store:       store,
        validator:   NewTaskValidator(),
        metrics:     NewEngineMetrics(),
        errorBuffer: ring.New(1000), // Keep last 1000 errors
    }
}

func (e *TaskEngine) SubmitTask(ctx context.Context, defID string, input map[string]interface{}) (*TaskInstance, error) {
    // Get task definition
    def, err := e.registry.GetDefinition(defID)
    if err != nil {
        return nil, fmt.Errorf("task definition not found: %v", err)
    }

    // Validate input
    if err := e.validator.ValidateInput(def, input); err != nil {
        return nil, fmt.Errorf("invalid input: %v", err)
    }

    // Create task instance
    task := &TaskInstance{
        ID:           generateTaskID(),
        DefinitionID: defID,
        Input:        input,
        Status: TaskStatus{
            Phase:    "pending",
            Progress: 0,
        },
        CreatedAt: time.Now(),
    }

    // Store task
    if err := e.store.SaveTask(task); err != nil {
        return nil, fmt.Errorf("failed to save task: %v", err)
    }

    // Start task processing
    go e.processTask(ctx, task)

    return task, nil
}

func (e *TaskEngine) processTask(ctx context.Context, task *TaskInstance) {
    // Update task status
    now := time.Now()
    task.Status.Phase = "running"
    task.StartedAt = &now
    e.store.UpdateTask(task)

    // Execute task
    if err := e.executor.ExecuteTask(ctx, task); err != nil {
        task.Status.Phase = "failed"
        task.Status.Message = err.Error()
        e.store.UpdateTask(task)
        return
    }

    // Update completion status
    completedAt := time.Now()
    task.Status.Phase = "completed"
    task.Status.Progress = 100
    task.CompletedAt = &completedAt
    e.store.UpdateTask(task)
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

// Circuit Breaker Implementation
type CircuitBreaker struct {
    mu           sync.RWMutex
    failures     int
    lastFailure  time.Time
    state        string // closed, open, half-open
    threshold    int
    resetTimeout time.Duration
}

func NewCircuitBreaker() *CircuitBreaker {
    return &CircuitBreaker{
        threshold:    5,
        resetTimeout: 30 * time.Second,
        state:       "closed",
    }
}

func (cb *CircuitBreaker) AllowRequest() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()

    if cb.state == "closed" {
        return true
    }

    if cb.state == "open" {
        if time.Since(cb.lastFailure) > cb.resetTimeout {
            cb.state = "half-open"
            return true
        }
        return false
    }

    return cb.state == "half-open"
}

// Add validation for content safety
type ContentValidator struct {
    toxicityThreshold float64
    client            *http.Client
}

func (v *ContentValidator) ValidateContent(content string) error {
    // Implement content moderation logic
    if containsSensitiveContent(content) {
        return ErrInappropriateContent
    }
    return nil
}

// Add structured logging
type Logger struct {
    logger *zap.Logger
}

func NewLogger() *Logger {
    config := zap.NewProductionConfig()
    logger, _ := config.Build()
    return &Logger{logger: logger}
}

func (l *Logger) LogTaskExecution(task *TaskInstance, duration time.Duration, err error) {
    l.logger.Info("task execution",
        zap.String("taskId", task.ID),
        zap.String("definitionId", task.DefinitionID),
        zap.Duration("duration", duration),
        zap.Error(err),
    )
}


// Task Validator
type TaskValidator struct {
    contentFilters map[string]ContentFilter
}

func NewTaskValidator() *TaskValidator {
    return &TaskValidator{
        contentFilters: map[string]ContentFilter{
            "profanity":        &ProfanityFilter{},
            "sensitive-topics": &SensitiveTopicsFilter{},
        },
    }
}

func (v *TaskValidator) ValidateInput(def *AITaskDefinition, input map[string]interface{}) error {
    // Validate against schema
    if err := validateSchema(def.InputSchema, input); err != nil {
        return err
    }

    // Apply content filters
    for _, filter := range def.Constraints.ContentFilters {
        if f, exists := v.contentFilters[filter]; exists {
            if err := f.Filter(input); err != nil {
                return fmt.Errorf("content filter %s failed: %v", filter, err)
            }
        }
    }

    return nil
}

// Content Filter Interface
type ContentFilter interface {
    Filter(content map[string]interface{}) error
}

type ProfanityFilter struct{}

func (f *ProfanityFilter) Filter(content map[string]interface{}) error {
    // Implementation for profanity filtering
    return nil
}

type SensitiveTopicsFilter struct{}

func (f *SensitiveTopicsFilter) Filter(content map[string]interface{}) error {
    // Implementation for sensitive topics filtering
    return nil
}

// Task Store Interface
type TaskStore interface {
    SaveTask(task *TaskInstance) error
    GetTask(id string) (*TaskInstance, error)
    UpdateTask(task *TaskInstance) error
    ListTasks(filter TaskFilter) ([]*TaskInstance, error)
}

type TaskFilter struct {
    DefinitionID string
    Status       string
    TimeRange    *TimeRange
}

type TimeRange struct {
    Start time.Time
    End   time.Time
}

// Helper Functions
func generateTaskID() string {
    return fmt.Sprintf("task-%d", time.Now().UnixNano())
}

func validateSchema(schema SchemaDefinition, data map[string]interface{}) error {
    // Schema validation implementation
    return nil
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
