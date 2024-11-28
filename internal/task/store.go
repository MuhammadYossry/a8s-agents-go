package engine

import (
    "sync"
    "time"
    "fmt"

    "github.com/MuhammadYossry/AgentNexus/internal/task/types"
)

// Task Store Interface
type TaskStore interface {
    SaveTask(task *TaskInstance) error
    GetTask(id string) (*TaskInstance, error)
    UpdateTask(task *TaskInstance) error
    ListTasks(filter TaskFilter) ([]*TaskInstance, error)
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

type TaskFilter struct {
    DefinitionID string
    Status       string
    TimeRange    *TimeRange
}

type TimeRange struct {
    Start time.Time
    End   time.Time
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