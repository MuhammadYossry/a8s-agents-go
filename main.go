package main

import (
    "context"
    "fmt"
    "sync"
    "time"
)

// Task Definition Core Types
type AITaskDefinition struct {
    TaskID      string         `json:"taskId"`
    Name        string         `json:"name"`
    Description string         `json:"description"`
    Version     string         `json:"version"`
    InputSchema  SchemaDefinition `json:"inputSchema"`
    OutputSchema SchemaDefinition `json:"outputSchema"`
    Capabilities []Capability    `json:"capabilities"`
    Constraints  TaskConstraints `json:"constraints"`
    Performance  PerformanceSpec `json:"performance"`
    Examples     []TaskExample   `json:"examples"`
    Tags         []string        `json:"tags"`
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

// Task Executor Interface
type TaskExecutor interface {
    ExecuteTask(ctx context.Context, task *TaskInstance) error
    ValidateInput(def *AITaskDefinition, input map[string]interface{}) error
    ValidateOutput(def *AITaskDefinition, output map[string]interface{}) error
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
    registry TaskRegistry
}

func NewDefaultTaskExecutor(registry TaskRegistry) *DefaultTaskExecutor {
    return &DefaultTaskExecutor{
        registry: registry,
    }
}

func (e *DefaultTaskExecutor) ExecuteTask(ctx context.Context, task *TaskInstance) error {
    // Get task definition
    def, err := e.registry.GetDefinition(task.DefinitionID)
    if err != nil {
        return fmt.Errorf("failed to get task definition: %v", err)
    }

    // For text summarization task
    if def.TaskID == "text-summarization-001" {
        return e.executeSummarizationTask(ctx, task, def)
    }

    return fmt.Errorf("unsupported task type: %s", def.TaskID)
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
    registry  TaskRegistry
    executor  TaskExecutor
    store     TaskStore
    validator *TaskValidator
}

func NewTaskEngine(registry TaskRegistry, executor TaskExecutor, store TaskStore) *TaskEngine {
    return &TaskEngine{
        registry:  registry,
        executor:  executor,
        store:     store,
        validator: NewTaskValidator(),
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
