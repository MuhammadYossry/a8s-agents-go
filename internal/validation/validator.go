package validation


import (
	"fmt"

    "github.com/MuhammadYossry/AgentNexus/internal/task/types"
)

// Task Validator
type TaskValidator struct {
    contentFilters map[string]ContentFilter
}

func NewTaskValidator() *TaskValidator {
    return &TaskValidator{
        contentFilters: map[string]ContentFilter{
            "profanity":        &ProfanityFilter{},
            "sensitive-topics": &SensitiveTopicsFilter{},
        },
    }
}

func validateSchema(schema SchemaDefinition, data map[string]interface{}) error {
    // Schema validation implementation
    return nil
}

func (v *TaskValidator) ValidateInput(def *AITaskDefinition, input map[string]interface{}) error {
    // Validate against schema
    if err := validateSchema(def.InputSchema, input); err != nil {
        return err
    }

    // Apply content filters
    for _, filter := range def.Constraints.ContentFilters {
        if f, exists := v.contentFilters[filter]; exists {
            if err := f.Filter(input); err != nil {
                return fmt.Errorf("content filter %s failed: %v", filter, err)
            }
        }
    }

    return nil
}