package broker

import (
    "sync"

    "github.com/MuhammadYossry/AgentNexus/internal/task/types"
)


type TaskQueue struct {
    mu       sync.RWMutex
    Tasks    chan *task.TaskInstance
    size     int
    capacity int
}

func NewTaskQueue(capacity int) *TaskQueue {
    return &TaskQueue{
        Tasks:    make(chan *task.TaskInstance, capacity),
        capacity: capacity,
    }
}

func (q *TaskQueue) Push(task *task.TaskInstance) error {
    q.mu.Lock()
    defer q.mu.Unlock()

    if q.size >= q.capacity {
        return ErrQueueFull
    }

    select {
    case q.Tasks <- task:
        q.size++
        return nil
    default:
        return ErrQueueFull
    }
}

func (q *TaskQueue) Size() int {
    q.mu.RLock()
    defer q.mu.RUnlock()
    return q.size
}

func (q *TaskQueue) Capacity() int {
    return q.capacity
}