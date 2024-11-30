package main

import (
	"time"
)

type AgentID string

type AgentCapability struct {
	AgentID   AgentID
	TaskTypes []string
	Resources map[string]int // e.g., "gpu": 1, "memory": 8
}
type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusComplete TaskStatus = "complete"
	TaskStatusFailed   TaskStatus = "failed"
)

type Task struct {
	ID           string
	Title        string
	Description  string // optional
	Type         string
	Capabilities []string // required capabilities
	Payload      []byte
	Status       TaskStatus
	RetryCount   int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TaskResult struct {
	TaskID     string
	Success    bool
	Output     []byte
	Error      string
	FinishedAt time.Time
}
