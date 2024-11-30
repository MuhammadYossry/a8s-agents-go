package main

import (
	"time"
)

type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusComplete TaskStatus = "complete"
	TaskStatusFailed   TaskStatus = "failed"
)

type Task struct {
	ID         string
	Type       string
	Payload    []byte
	Status     TaskStatus
	RetryCount int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type TaskResult struct {
	TaskID     string
	Success    bool
	Output     []byte
	Error      string
	FinishedAt time.Time
}
