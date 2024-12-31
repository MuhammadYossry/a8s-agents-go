package types

import (
	"context"
	"time"
)

type TaskStatus string
type TaskPath []string
type WorkFlowID string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusComplete TaskStatus = "complete"
	TaskStatusFailed   TaskStatus = "failed"
)

type contextKey string

const (
	RawAgentsDataKey        contextKey = "agents_definations"
	TaskExtractionResultKey contextKey = "task_extraction_result"
	AgentsMarkDownKey       contextKey = "agents_definations_markdown"
)

type Task struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
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
type TaskExecutorConfig struct {
	AgentDefinition *AgentDefinition
	PayloadAgent    PayloadAgent
	HTTPTimeout     time.Duration
}

type TaskRequirement struct {
	SkillPath  TaskPath               `json:"path"`       // e.g. ["Development", "Backend", "Python", "CodeGeneration"]
	Action     string                 `json:"action"`     // e.g. "generateCode"
	Parameters map[string]interface{} `json:"parameters"` // Additional parameters for matching
}

type WorkFlowCapability struct {
	WorkFlowID   WorkFlowID
	Capabilities []Capability
	Resources    map[string]int
}

type Executor interface {
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
}

// Broker defines the message broker interface
type Broker interface {
	Publish(ctx context.Context, topic string, task *Task) error
	Subscribe(ctx context.Context, topic string) (<-chan *Task, error)
	Unsubscribe(ctx context.Context, topic string, ch <-chan *Task) error
	Close() error
}

// TaskRouter interface defines the contract for routing tasks
type TaskRouter interface {
	RouteTask(ctx context.Context, task *Task) error
}

// MetricsCollector interface defines the metrics collection contract
type MetricsCollector interface {
	RecordTaskStart(taskID string)
	RecordTaskComplete(requirements TaskRequirement, taskID string)
	RecordTaskError(requirements TaskRequirement, err error)
	RecordRoutingSuccess(requirements TaskRequirement, agentID string)
	RecordRoutingFailure(requirements TaskRequirement, reason string)
	GetMetrics(requirements TaskRequirement) *MetricsData
	GetAllMetrics() map[MetricsKey]*MetricsData
	ResetMetrics()
}

// types.MetricsCollector defines the interface for collecting metrics
type MetricsData struct {
	TasksCompleted        int
	TasksFailed           int
	RoutingFailures       int
	RoutingSuccesses      int
	LastError             error
	LastErrorTime         time.Time
	TotalProcessingTime   time.Duration
	AverageProcessingTime time.Duration
	ProcessingTimes       []time.Duration
}

// MetricsKey uniquely identifies a metric category
type MetricsKey struct {
	SkillPath string // Dot-separated path
	Action    string
}
