package engine

import (
    "context"
    "time"

    "github.com/Relax-N-Tax/AgentNexus/internal/task/types"
    "github.com/Relax-N-Tax/AgentNexus/internal/validation"
    "github.com/Relax-N-Tax/AgentNexus/pkg/metrics"

    "container/ring"
)

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

// Helper Functions
func generateTaskID() string {
    return fmt.Sprintf("task-%d", time.Now().UnixNano())
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