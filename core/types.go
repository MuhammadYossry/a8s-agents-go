package core

import (
	"context"
	"encoding/json"
)

// Capability interface abstraction
type CapabilityExecutor interface {
	Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
	ValidateInput(input json.RawMessage) error
	GetMetadata() map[string]interface{}
}
