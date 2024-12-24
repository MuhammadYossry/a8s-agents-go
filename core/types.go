package core

import (
	"context"
	"encoding/json"
	"time"
)

type AgentID string
type AgentType string
type WorkFlowID string
type TaskStatus string
type TaskPath []string

// Agenter interface defines the core functionality required by all agents
type Agenter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
	GetCapabilities() []AgentCapability
}

type AgentCapability struct {
	AgentID      AgentID
	Capabilities []Capability
	Actions      []Action
	Resources    map[string]int
}

// Capability interface abstraction
type CapabilityExecutor interface {
	Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
	ValidateInput(input json.RawMessage) error
	GetMetadata() map[string]interface{}
}

type AgentConfig struct {
	Agents []AgentDefinition `json:"agents"`
}

type AgentDefinition struct {
	ID           string       `json:"id"`
	Type         string       `json:"type"`
	Description  string       `json:"description"`
	BaseURL      string       `json:"baseURL"`
	Capabilities []Capability `json:"capabilities"`
	Actions      []Action     `json:"actions"`
}

type Action struct {
	Name         string       `json:"name"`
	Path         string       `json:"path"`
	Method       string       `json:"method"`
	InputSchema  SchemaConfig `json:"inputSchema"`
	OutputSchema SchemaConfig `json:"outputSchema"`
}

type WorkFlowCapability struct {
	WorkFlowID   WorkFlowID
	Capabilities []Capability
	Resources    map[string]int
}

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusComplete TaskStatus = "complete"
	TaskStatusFailed   TaskStatus = "failed"
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
type TaskRequirement struct {
	SkillPath  TaskPath               `json:"path"`       // e.g. ["Development", "Backend", "Python", "CodeGeneration"]
	Action     string                 `json:"action"`     // e.g. "generateCode"
	Parameters map[string]interface{} `json:"parameters"` // Additional parameters for matching
}

type SchemaConfig struct {
	Type                 string              `json:"type"`
	Required             []string            `json:"required,omitempty"`
	Properties           map[string]Property `json:"properties,omitempty"`
	AdditionalProperties interface{}         `json:"additionalProperties,omitempty"`
	Description          string              `json:"description,omitempty"`
	Example              interface{}         `json:"example,omitempty"`
	Examples             []interface{}       `json:"examples,omitempty"`
}

type Property struct {
	Type                 string              `json:"type"`
	Formats              []string            `json:"formats,omitempty"`
	MaxSize              string              `json:"max_size,omitempty"`
	Enum                 []string            `json:"enum,omitempty"`
	Default              interface{}         `json:"default,omitempty"`
	Items                *Property           `json:"items,omitempty"`      // For array types
	Properties           map[string]Property `json:"properties,omitempty"` // For object types
	Required             []string            `json:"required,omitempty"`
	MinimumItems         int                 `json:"minItems,omitempty"`
	MaximumItems         int                 `json:"maxItems,omitempty"`
	Pattern              string              `json:"pattern,omitempty"`
	Format               string              `json:"format,omitempty"`
	AdditionalProperties interface{}         `json:"additionalProperties,omitempty"`
}
