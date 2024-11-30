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
	// Simple implementation that just marks the task as complete
	return &TaskResult{
		TaskID:     task.ID,
		Success:    true,
		FinishedAt: time.Now(),
	}, nil
}
