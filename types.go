package main

import (
	"time"
)

type AgentID string
type AgentType string
type AgentCapability struct {
	AgentID      AgentID
	Capabilities []Capability
	Actions      []Action
	Resources    map[string]int
}
type WorkFlowID string
type WorkFlowCapability struct {
	WorkFlowID   WorkFlowID
	Capabilities []Capability
	Resources    map[string]int
}
type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusComplete TaskStatus = "complete"
	TaskStatusFailed   TaskStatus = "failed"
)

type TaskPath []string
type TaskRequirement struct {
	SkillPath  TaskPath               `json:"path"`       // e.g. ["Development", "Backend", "Python", "CodeGeneration"]
	Action     string                 `json:"action"`     // e.g. "generateCode"
	Parameters map[string]interface{} `json:"parameters"` // Additional parameters for matching
}

type Task struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Requirements TaskRequirement `json:"requirements"`
	Payload      []byte          `json:"payload"`
	Status       TaskStatus      `json:"status"`
	RetryCount   int             `json:"retryCount"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type TaskResult struct {
	TaskID     string
	Success    bool
	Output     []byte
	Error      string
	FinishedAt time.Time
}
