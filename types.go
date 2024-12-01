package main

import (
	"time"
)

type AgentID string
type WorkFlowID string
type AgentCapability struct {
	AgentID   AgentID
	TaskTypes []string
	// Skills map is now organized by task type
	SkillsByType map[string][]string // map[TaskType][]Skills
	Resources    map[string]int
}
type WorkFlowCapability struct {
	WorkFlowID   WorkFlowID
	TaskTypes    []string
	SkillsByType map[string][]string
	Resources    map[string]int
}
type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusComplete TaskStatus = "complete"
	TaskStatusFailed   TaskStatus = "failed"
)

type Task struct {
	ID             string
	Title          string
	Description    string // optional
	Type           string
	SkillsRequired []string // required skills
	Payload        []byte
	Status         TaskStatus
	RetryCount     int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TaskResult struct {
	TaskID     string
	Success    bool
	Output     []byte
	Error      string
	FinishedAt time.Time
}
