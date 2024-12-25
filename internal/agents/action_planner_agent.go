// action_planner_agent.go
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

// ActionPlannerAgent determines the most suitable action for a given task
type ActionPlannerAgent struct {
	llmClient *LLMClient
	promptMgr *PromptManager
}

type ActionPlannerConfig struct {
	LLMConfig LLMConfig
}

var (
	actionPlannerInstance *ActionPlannerAgent
	actionPlannerOnce     sync.Once
)

const actionPlannerPromptTemplate = `System Message:
You are a technical assistant that analyzes software development requirements and determines the most suitable action to execute.

Task Information:
Title: {{.task.Title}}
Description: {{.task.Description}}
Required Skill Path: {{.task.Requirements.SkillPath}}
Required Action: {{.task.Requirements.Action}}
{{- range $key, $value := .task.Requirements.Parameters}}
Parameter - {{$key}}: {{$value}}
{{- end}}

Available Actions:
{{- range .actions}}
{{.Name}}:
  - Path: {{.Path}}
  - Method: {{.Method}}
  {{- if .InputSchema}}
  - Input Schema:
    {{- range $key, $value := .InputSchema.Properties}}
    * {{$key}}
    {{- end}}
  {{- end}}
{{- end}}

Please analyze and provide your response in the following JSON structure:

{
  "selectedAction": string,            // Name of the chosen action
  "confidence": number,               // 0-1 score of how well the action matches
  "reasoning": {
    "primary_reason": string,         // Main reason for selecting this action
    "alignment_points": [             // List of specific matching points
      string,
      ...
    ],
    "potential_concerns": [           // Any potential issues to consider
      string,
      ...
    ]
  },
  "implementation": {
    "required_parameters": {          // Parameters that must be provided
      parameter_name: parameter_value,
      ...
    },
    "recommended_optional_parameters": {  // Optional parameters that would be beneficial
      parameter_name: parameter_value,
      ...
    }
  },
  "validation": {
    "framework_compatible": boolean,   // Confirms framework compatibility
    "skill_path_supported": boolean,  // Confirms skill path is supported
    "missing_requirements": [         // Any required information not provided in task
      string,
      ...
    ]
  }
}`

// GetActionPlannerAgent returns the singleton instance of ActionPlannerAgent
func GetActionPlannerAgent(ctx context.Context, config types.InternalAgentConfig) (*ActionPlannerAgent, error) {
	actionPlannerOnce.Do(func() {
		actionPlannerInstance = initializeActionPlannerAgent(config)
	})
	return actionPlannerInstance, nil
}

func initializeActionPlannerAgent(config types.InternalAgentConfig) *ActionPlannerAgent {
	llmClient := NewLLMClient(&LLMConfig{
		BaseURL: config.LLMConfig.BaseURL,
		APIKey:  config.LLMConfig.APIKey,
		Model:   config.LLMConfig.Model,
		Timeout: config.LLMConfig.Timeout,
	})

	promptMgr := NewPromptManager()
	if err := promptMgr.RegisterTemplate("actionPlannerPrompt", actionPlannerPromptTemplate); err != nil {
		panic(fmt.Sprintf("Failed to register action planner prompt template: %v", err))
	}

	return &ActionPlannerAgent{
		llmClient: llmClient,
		promptMgr: promptMgr,
	}
}

// PlanAction determines the most suitable action for the given task
func (a *ActionPlannerAgent) PlanAction(ctx context.Context, task *types.Task, availableActions []types.Action) (*types.ActionPlan, error) {
	// Prepare prompt data
	promptData := map[string]interface{}{
		"task":    task,
		"actions": availableActions,
	}

	// Generate prompt
	prompt, err := a.promptMgr.GeneratePrompt("actionPlannerPrompt", promptData)
	fmt.Println("Action Planner Prompt")
	fmt.Println(prompt)
	if err != nil {
		return nil, fmt.Errorf("generating prompt: %w", err)
	}

	// For demonstration, using a sample response
	completion := `{
        "selectedAction": "generateCode",
        "confidence": 0.95,
        "reasoning": {
            "primary_reason": "Task explicitly requires new code generation for a REST API endpoint",
            "alignment_points": [
                "Supports FastAPI framework requirement",
                "Matches Python/Backend/CodeGeneration skill path",
                "Provides full testing and documentation capabilities"
            ],
            "potential_concerns": [
                "May need additional specification of required functions"
            ]
        },
        "implementation": {
            "required_parameters": {
                "language": "Python",
                "framework": "FastAPI",
                "description": "Create a REST API endpoint"
            },
            "recommended_optional_parameters": {
                "documentationLevel": "detailed",
                "includeTests": true
            }
        },
        "validation": {
            "framework_compatible": true,
            "skill_path_supported": true,
            "missing_requirements": []
        }
    }`

	// Parse the response into ActionPlan
	var actionPlan types.ActionPlan
	if err := json.Unmarshal([]byte(completion), &actionPlan); err != nil {
		return nil, fmt.Errorf("parsing action plan: %w", err)
	}

	return &actionPlan, nil
}
