package types

import (
	"time"
)

type AgentID string
type AgentType string
type TaskStatus string
type TaskPath []string

type InternalAgentConfig struct {
	LLMConfig struct {
		BaseURL string
		APIKey  string
		Model   string
		Timeout time.Duration
	}
}

type AgentCapability struct {
	AgentID      AgentID
	Capabilities []Capability
	Actions      []Action
	Resources    map[string]int
}

type Capability struct {
	SkillPath []string               `json:"skillPath"`
	Level     string                 `json:"level"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type Action struct {
	Name         string `json:"name"`
	BaseURL      string
	Path         string       `json:"path"`
	Method       string       `json:"method"`
	InputSchema  SchemaConfig `json:"inputSchema"`
	OutputSchema SchemaConfig `json:"outputSchema"`
}

type SchemaConfig struct {
	Type                 string                  `json:"type"`
	Required             []string                `json:"required,omitempty"`
	Properties           map[string]Property     `json:"properties,omitempty"` // Changed from SchemaProperty to Property
	Description          string                  `json:"description,omitempty"`
	Title                string                  `json:"title,omitempty"`
	Defs                 map[string]SchemaConfig `json:"$defs,omitempty"`
	Ref                  string                  `json:"$ref,omitempty"`
	AdditionalProperties *Property               `json:"additionalProperties,omitempty"` // Changed from SchemaProperty to Property
	Default              interface{}             `json:"default,omitempty"`
	Examples             []interface{}           `json:"examples,omitempty"` // Added Examples field
	Example              interface{}             `json:"example,omitempty"`  // Added Example field
}
type Property struct {
	Type                 string              `json:"type"`
	Title                string              `json:"title,omitempty"`
	Description          string              `json:"description,omitempty"`
	Format               string              `json:"format,omitempty"`
	Pattern              string              `json:"pattern,omitempty"` // Added Pattern field
	Default              interface{}         `json:"default,omitempty"`
	Enum                 []string            `json:"enum,omitempty"`
	Const                string              `json:"const,omitempty"`
	Items                *Property           `json:"items,omitempty"`
	Properties           map[string]Property `json:"properties,omitempty"`
	Required             []string            `json:"required,omitempty"`
	MinimumItems         int                 `json:"minItems,omitempty"`
	MaximumItems         int                 `json:"maxItems,omitempty"`
	Minimum              *float64            `json:"minimum,omitempty"`
	Maximum              *float64            `json:"maximum,omitempty"`
	AnyOf                []Property          `json:"anyOf,omitempty"`
	AllOf                []Property          `json:"allOf,omitempty"`
	OneOf                []Property          `json:"oneOf,omitempty"`
	Ref                  string              `json:"$ref,omitempty"`
	AdditionalProperties *Property           `json:"additionalProperties,omitempty"`
	Examples             []interface{}       `json:"examples,omitempty"`
	Example              interface{}         `json:"example,omitempty"`
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

// types/action_planner.go

type ActionPlan struct {
	SelectedAction string               `json:"selectedAction"`
	Confidence     float64              `json:"confidence"`
	Reasoning      ActionPlanReasoning  `json:"reasoning"`
	Implementation ActionImplementation `json:"implementation"`
	Validation     ActionValidation     `json:"validation"`
}

type ActionPlanReasoning struct {
	PrimaryReason     string   `json:"primary_reason"`
	AlignmentPoints   []string `json:"alignment_points"`
	PotentialConcerns []string `json:"potential_concerns"`
}

type ActionImplementation struct {
	RequiredParameters        map[string]interface{} `json:"required_parameters"`
	RecommendedOptionalParams map[string]interface{} `json:"recommended_optional_parameters"`
}

type ActionValidation struct {
	FrameworkCompatible bool     `json:"framework_compatible"`
	SkillPathSupported  bool     `json:"skill_path_supported"`
	MissingRequirements []string `json:"missing_requirements"`
}
