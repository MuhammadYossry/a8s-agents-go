package types

import (
	"context"
	"time"
)

type AgentID string
type AgentType string

type AgentDefinition struct {
	ID           string       `json:"id"`
	Type         string       `json:"type"`
	Description  string       `json:"description"`
	BaseURL      string       `json:"baseURL"`
	Capabilities []Capability `json:"capabilities"`
	Actions      []Action     `json:"actions"`
}

type AgentConfig struct {
	Agents []AgentDefinition `json:"agents"`
}

type InternalAgentConfig struct {
	LLMConfig struct {
		BaseURL string
		APIKey  string
		Model   string
		Timeout time.Duration
	}
	AgentsMDFormatter MarkdownFormatter
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
	Name         string       `json:"name"`
	BaseURL      string       `json:"baseURL"`
	ActionType   string       `json:"type"`
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

// action_planner types

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

// AgentMatch represents the selected agent and action details
type AgentMatch struct {
	AgentID      string       `json:"agentId"`
	Action       string       `json:"action"`
	Confidence   float64      `json:"confidence"`
	MatchDetails MatchDetails `json:"matchDetails"`
	Reasoning    string       `json:"reasoning"`
}

// MatchDetails contains the detailed scoring of the match
type MatchDetails struct {
	PathMatchScore float64 `json:"pathMatchScore"`
	FrameworkScore float64 `json:"frameworkScore"`
	ActionScore    float64 `json:"actionScore"`
	VersionScore   float64 `json:"versionScore"`
}

// Alternative represents other potential agent matches
type Alternative struct {
	AgentID    string  `json:"agentId"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// MatchResult represents the output of the routing decision
type MatchResult struct {
	Matched      bool          `json:"matched"`
	Match        *AgentMatch   `json:"match,omitempty"`
	Alternatives []Alternative `json:"alternatives,omitempty"`
	Error        string        `json:"error,omitempty"`
}

type AgentLoader interface {
	LoadAgents(ctx context.Context, filepath string, config InternalAgentConfig) (context.Context, []Agenter, error)
}

type AgentFactory interface {
	GetPayloadAgent(ctx context.Context, config InternalAgentConfig) (PayloadAgent, error)
	GetTaskRoutingAgent(config InternalAgentConfig) (TaskRoutingAgent, error)
	GetTaskExtractionAgent(config InternalAgentConfig) (TaskExtractionAgent, error)
}

// Internal PayloadAgent interface defines methods for handling task payloads
type PayloadAgent interface {
	GeneratePayload(ctx context.Context, task *Task, action Action) ([]byte, error)
	GeneratePayloadWithRetry(ctx context.Context, task *Task, action Action) ([]byte, error)
}

// Agenter interface defines the core functionality required by all external agents
type Agenter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
	GetCapabilities() []AgentCapability
}

// TaskRoutingAgent interface defines the contract for routing tasks
type TaskRoutingAgent interface {
	FindMatchingAgent(ctx context.Context, task *Task) (*ActionPlan, error)
}

// TaskExtractionAgent interface defines the contract for extracting tasks
type TaskExtractionAgent interface {
	ExtractTaskWithRetry(ctx context.Context, query string) (context.Context, error)
}

type AgentDefConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}
