package core

import (
	"context"
	"encoding/json"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

type WorkFlowID string

// Agenter interface defines the core functionality required by all agents
type Agenter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Execute(ctx context.Context, task *types.Task) (*types.TaskResult, error)
	GetCapabilities() []types.AgentCapability
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
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Description  string             `json:"description"`
	BaseURL      string             `json:"baseURL"`
	Capabilities []types.Capability `json:"capabilities"`
	Actions      []types.Action     `json:"actions"`
}

type WorkFlowCapability struct {
	WorkFlowID   WorkFlowID
	Capabilities []types.Capability
	Resources    map[string]int
}
