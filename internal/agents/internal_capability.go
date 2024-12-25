package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

// CapabilityExecutor defines the common interface for all capabilities
type CapabilityExecutor interface {
	Execute(ctx context.Context, task *types.Task) (*types.TaskResult, error)
	GetSkillPath() []string
	GetLevel() string
	GetMetadata() map[string]interface{}
	ValidateRequirements(req types.TaskRequirement) error
}

// InternalCapability extends core.Capability for internal agent use
type InternalCapability struct {
	types.Capability // Embed core.Capability
	Handler          CapabilityHandler
	ValidationRules  []ValidationRule
	lastExecutedAt   time.Time
	executionCount   int64
}

// CapabilityHandler defines the execution function signature
type CapabilityHandler func(ctx context.Context, payload []byte) ([]byte, error)

// ValidationRule defines a single validation rule
type ValidationRule struct {
	Name     string
	Validate func(req types.TaskRequirement) error
}

// NewInternalCapability creates a new internal capability
func NewInternalCapability(config InternalCapabilityConfig) *InternalCapability {
	return &InternalCapability{
		Capability: types.Capability{
			SkillPath: config.SkillPath,
			Level:     config.Level,
			Metadata: map[string]interface{}{
				"created_at":  time.Now().Unix(),
				"is_internal": true,
				"name":        config.Name,
				"description": config.Description,
				"version":     config.Version,
			},
		},
		Handler:         config.Handler,
		ValidationRules: config.ValidationRules,
	}
}

// InternalCapabilityConfig holds configuration for creating a new capability
type InternalCapabilityConfig struct {
	Name            string
	Description     string
	Version         string
	SkillPath       []string
	Level           string
	Handler         CapabilityHandler
	ValidationRules []ValidationRule
}

// Implement CapabilityExecutor interface
func (ic *InternalCapability) Execute(ctx context.Context, task *types.Task) (*types.TaskResult, error) {
	if err := ic.ValidateRequirements(task.Requirements); err != nil {
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      fmt.Sprintf("validation failed: %v", err),
			FinishedAt: time.Now(),
		}, nil
	}

	output, err := ic.Handler(ctx, task.Payload)
	if err != nil {
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      err.Error(),
			FinishedAt: time.Now(),
		}, nil
	}

	ic.lastExecutedAt = time.Now()
	ic.executionCount++

	return &types.TaskResult{
		TaskID:     task.ID,
		Success:    true,
		Output:     output,
		FinishedAt: time.Now(),
	}, nil
}

func (ic *InternalCapability) ValidateRequirements(req types.TaskRequirement) error {
	// Validate skill path
	if !ic.matchesSkillPath(req.SkillPath) {
		return fmt.Errorf("skill path mismatch: required %v, have %v", req.SkillPath, ic.SkillPath)
	}

	// Run all validation rules
	for _, rule := range ic.ValidationRules {
		if err := rule.Validate(req); err != nil {
			return fmt.Errorf("%s validation failed: %w", rule.Name, err)
		}
	}

	return nil
}

func (ic *InternalCapability) GetSkillPath() []string {
	return ic.SkillPath
}

func (ic *InternalCapability) GetLevel() string {
	return ic.Level
}

func (ic *InternalCapability) GetMetadata() map[string]interface{} {
	metadata := ic.Metadata
	metadata["last_executed_at"] = ic.lastExecutedAt.Unix()
	metadata["execution_count"] = ic.executionCount
	return metadata
}

// Helper methods
func (ic *InternalCapability) matchesSkillPath(reqPath []string) bool {
	if len(reqPath) > len(ic.SkillPath) {
		return false
	}

	for i, skill := range reqPath {
		if i >= len(ic.SkillPath) || ic.SkillPath[i] != skill {
			return false
		}
	}
	return true
}

// Example usage:
func ExampleCreateCapability() *InternalCapability {
	return NewInternalCapability(InternalCapabilityConfig{
		Name:        "CodeGeneration",
		Description: "Generates code based on requirements",
		Version:     "1.0.0",
		SkillPath:   []string{"Development", "Backend", "CodeGeneration"},
		Level:       "advanced",
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			// Implementation
			return payload, nil
		},
		ValidationRules: []ValidationRule{
			{
				Name: "LanguageValidation",
				Validate: func(req types.TaskRequirement) error {
					lang, ok := req.Parameters["language"]
					if !ok {
						return fmt.Errorf("language parameter is required")
					}
					supportedLangs := []string{"python", "go", "javascript"}
					for _, supported := range supportedLangs {
						if lang == supported {
							return nil
						}
					}
					return fmt.Errorf("unsupported language: %v", lang)
				},
			},
		},
	})
}
