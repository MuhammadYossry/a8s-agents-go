package main

import (
	"context"
	"time"
)

type Executor interface {
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
}

type TaskExecutor struct{}

func NewTaskExecutor() *TaskExecutor {
	return &TaskExecutor{}
}

func (e *TaskExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	// Simulate some work being done
	time.Sleep(2 * time.Second)

	return &TaskResult{
		TaskID:     task.ID,
		Success:    true,
		FinishedAt: time.Now(),
	}, nil
}
