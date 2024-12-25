// payload_agent.go
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

// PayloadAgent generates valid JSON payloads for different actions based on their schemas
type PayloadAgent struct {
	llmClient *LLMClient
	promptMgr *PromptManager
}

var (
	instance *PayloadAgent
	once     sync.Once
)

const payloadPromptTemplate = `System Message:
You are an API request generator. Generate valid JSON payloads that match the schema exactly. Return only the JSON payload without explanations.

Task Information:
- Type: {{.task.Type}}
- Description: {{.task.Description}}
- Skills Required: {{.task.Requirements.SkillPath}}
{{- range $key, $value := .task.Requirements.Parameters}}
- {{$key}}: {{$value}}
{{- end}}
- Additional Info: {{.task.Title}}

Required Request Schema:
{{.schemaStr}}

Example Valid Request(Ign):
{{.exampleStr}}

Generate a valid JSON payload following the schema and example structure, incorporating the task requirements. The response should be only the JSON payload, with no additional text or explanations.`

// GetPayloadAgent returns the singleton instance of PayloadAgent
func GetPayloadAgent(ctx context.Context, config types.PayloadAgentConfig) (*PayloadAgent, error) {
	once.Do(func() {
		instance = initializePayloadAgent(config)
	})
	return instance, nil
}

func initializePayloadAgent(config types.PayloadAgentConfig) *PayloadAgent {
	llmClient := NewLLMClient(&LLMConfig{
		BaseURL: config.LLMConfig.BaseURL,
		APIKey:  config.LLMConfig.APIKey,
		Model:   config.LLMConfig.Model,
		Timeout: config.LLMConfig.Timeout,
	})

	promptMgr := NewPromptManager()
	if err := promptMgr.RegisterTemplate("payloadPrompt", payloadPromptTemplate); err != nil {
		panic(fmt.Sprintf("Failed to register payload prompt template: %v", err))
	}

	return &PayloadAgent{
		llmClient: llmClient,
		promptMgr: promptMgr,
	}
}

// GeneratePayload generates a valid JSON payload for the given action based on the task context
func (a *PayloadAgent) GeneratePayload(ctx context.Context, task *types.Task, action types.Action) ([]byte, error) {
	// Convert input schema to string representation
	schemaBytes, err := json.MarshalIndent(action.InputSchema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling input schema: %w", err)
	}

	// Get example if available, or create empty object
	var exampleBytes []byte
	if len(action.InputSchema.Examples) > 0 {
		exampleBytes, err = json.MarshalIndent(action.InputSchema.Examples[0], "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshaling example: %w", err)
		}
	} else {
		exampleBytes = []byte("No examples provided")
	}

	// Prepare prompt data
	promptData := map[string]interface{}{
		"task":       task,
		"schemaStr":  string(schemaBytes),
		"exampleStr": string(exampleBytes),
	}

	// Generate prompt
	prompt, err := a.promptMgr.GeneratePrompt("payloadPrompt", promptData)
	if err != nil {
		return nil, fmt.Errorf("generating prompt: %w", err)
	}
	fmt.Println(prompt)

	completion := `{
  "codeRequirements": "Create a FastAPI endpoint with CRUD operations for a user resource.",
  "documentationLevel": "detailed",
  "includeTests": true,
  "styleGuide": "PEP 8"
}`

	// Validate the response is valid JSON
	var jsonResponse interface{}
	if err := json.Unmarshal([]byte(completion), &jsonResponse); err != nil {
		return nil, fmt.Errorf("invalid JSON response from LLM: %w", err)
	}

	return []byte(completion), nil
}
