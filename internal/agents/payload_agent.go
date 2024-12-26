// payload_agent.go
package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

type PayloadAgent struct {
	llmClient *LLMClient
	promptMgr *PromptManager
}

var (
	instance *PayloadAgent
	once     sync.Once
)

const payloadPromptTemplate = `System Message:
You are a JSON schema expert. Create a payload that exactly matches this schema:

Schema Details:
{{.schemaStr}}

Task Context:
{{- if .taskContext}}
{{.taskContext}}
{{- end}}

Example:
{{.exampleStr}}

IMPORTANT:
- Return ONLY a valid JSON object
- All values must match their defined types exactly
- Arrays must contain items of the correct type
- Include all required fields
- Follow any format/pattern requirements
- Respect enum value restrictions

Response format: JSON object only.`

func GetPayloadAgent(ctx context.Context, config types.InternalAgentConfig) (*PayloadAgent, error) {
	once.Do(func() {
		instance = initializePayloadAgent(config)
	})
	return instance, nil
}

func initializePayloadAgent(config types.InternalAgentConfig) *PayloadAgent {
	llmClient := NewLLMClient(&LLMConfig{
		Provider:      Qwen,
		BaseURL:       config.LLMConfig.BaseURL,
		APIKey:        config.LLMConfig.APIKey,
		Model:         config.LLMConfig.Model,
		Timeout:       90 * time.Second,
		Debug:         true,
		SystemMessage: "You are a specialized JSON generator that produces exact schema-compliant payloads.",
		Options: map[string]interface{}{
			"temperature": 0.7,
			"top_p":       0.8,
		},
	})

	promptMgr := NewPromptManager()
	if err := promptMgr.RegisterTemplate("payloadPrompt", payloadPromptTemplate); err != nil {
		panic(fmt.Sprintf("Failed to register payload prompt template: %v", err))
	}
	if err := promptMgr.RegisterTemplate("payloadFixPrompt", payloadFixPromptTemplate); err != nil {
		panic(fmt.Sprintf("Failed to register payload fix prompt template: %v", err))
	}

	return &PayloadAgent{
		llmClient: llmClient,
		promptMgr: promptMgr,
	}
}

func (a *PayloadAgent) GeneratePayload(ctx context.Context, task *types.Task, action types.Action) ([]byte, error) {
	schemaInfo := formatSchema(action.InputSchema)
	example := generateExample(action.InputSchema, task)
	taskContext := generateTaskContext(task)

	promptData := map[string]interface{}{
		"schemaStr":   schemaInfo,
		"exampleStr":  string(example),
		"taskContext": taskContext,
	}

	prompt, err := a.promptMgr.GeneratePrompt("payloadPrompt", promptData)
	if err != nil {
		return nil, fmt.Errorf("generating prompt: %w", err)
	}

	if Debug {
		log.Printf("Generated prompt:\n%s\n", prompt)
	}

	completion, err := getCompletionWithRetries(ctx, a.llmClient, prompt)
	if err != nil {
		return nil, err
	}

	return validateAndFormatJSON(completion, action.InputSchema)
}

func formatSchema(schema types.SchemaConfig) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Type: %s\n", schema.Type))
	if schema.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n", schema.Description))
	}

	if len(schema.Required) > 0 {
		b.WriteString("\nRequired Fields:\n")
		for _, field := range schema.Required {
			prop := schema.Properties[field]
			formatProperty(&b, field, prop, true)
		}
	}

	for name, prop := range schema.Properties {
		if !contains(schema.Required, name) {
			formatProperty(&b, name, prop, false)
		}
	}

	return b.String()
}

func formatProperty(b *strings.Builder, name string, prop types.Property, required bool) {
	indent := "  "
	var marker string
	if required {
		marker = "*"
	} else {
		marker = "-"
	}

	b.WriteString(fmt.Sprintf("%s %s:\n", marker, name))
	b.WriteString(fmt.Sprintf("%sType: %s\n", indent, prop.Type))

	if len(prop.Enum) > 0 {
		b.WriteString(fmt.Sprintf("%sAllowed values: [%s]\n", indent, strings.Join(prop.Enum, ", ")))
	}

	if prop.Format != "" {
		b.WriteString(fmt.Sprintf("%sFormat: %s\n", indent, prop.Format))
	}

	if prop.Pattern != "" {
		b.WriteString(fmt.Sprintf("%sPattern: %s\n", indent, prop.Pattern))
	}

	if prop.Type == "array" && prop.Items != nil {
		b.WriteString(fmt.Sprintf("%sArray items:\n", indent))
		b.WriteString(fmt.Sprintf("%s  Type: %s\n", indent, prop.Items.Type))
		if len(prop.Items.Enum) > 0 {
			b.WriteString(fmt.Sprintf("%s  Allowed values: [%s]\n", indent, strings.Join(prop.Items.Enum, ", ")))
		}
	}

	if prop.Type == "object" && len(prop.Properties) > 0 {
		b.WriteString(fmt.Sprintf("%sObject properties:\n", indent))
		for subName, subProp := range prop.Properties {
			isRequired := contains(prop.Required, subName)
			formatProperty(b, subName, subProp, isRequired)
		}
	}
}

func generateExample(schema types.SchemaConfig, task *types.Task) []byte {
	if len(schema.Examples) > 0 {
		if example, err := json.MarshalIndent(schema.Examples[0], "", "  "); err == nil {
			return example
		}
	}

	example := generateExampleForSchema(schema, task)
	exampleBytes, _ := json.MarshalIndent(example, "", "  ")
	return exampleBytes
}

func generateExampleForSchema(schema types.SchemaConfig, task *types.Task) interface{} {
	if schema.Type == "object" {
		obj := make(map[string]interface{})
		for name, prop := range schema.Properties {
			obj[name] = generateExampleForProperty(prop, task)
		}
		return obj
	}
	return nil
}

func generateExampleForProperty(prop types.Property, task *types.Task) interface{} {
	switch prop.Type {
	case "string":
		if len(prop.Enum) > 0 {
			return prop.Enum[0]
		}
		if prop.Default != nil {
			return prop.Default
		}
		return "example"

	case "array":
		if prop.Items != nil {
			return []interface{}{generateExampleForProperty(*prop.Items, task)}
		}
		return []interface{}{}

	case "object":
		obj := make(map[string]interface{})
		for name, subProp := range prop.Properties {
			obj[name] = generateExampleForProperty(subProp, task)
		}
		return obj

	case "boolean":
		if prop.Default != nil {
			return prop.Default
		}
		return true

	case "integer", "number":
		if prop.Default != nil {
			return prop.Default
		}
		return 0
	}

	return nil
}

func generateTaskContext(task *types.Task) string {
	var b strings.Builder
	b.WriteString("Requirements:\n")

	if task.Description != "" {
		b.WriteString(fmt.Sprintf("- Description: %s\n", task.Description))
	}

	for key, value := range task.Requirements.Parameters {
		b.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
	}

	return b.String()
}

func getCompletionWithRetries(ctx context.Context, client *LLMClient, prompt string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		completion, err := client.CreateChatCompletion(ctx, prompt)
		if err == nil {
			return completion, nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
	}
	return "", fmt.Errorf("failed after 3 attempts: %w", lastErr)
}

func validateAndFormatJSON(input string, schema types.SchemaConfig) ([]byte, error) {
	input = strings.Trim(strings.TrimSpace(input), "`")

	var jsonData interface{}
	if err := json.Unmarshal([]byte(input), &jsonData); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if err := validateAgainstSchema(jsonData, schema); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	return json.MarshalIndent(jsonData, "", "  ")
}

func validateAgainstSchema(data interface{}, schema types.SchemaConfig) error {
	if schema.Type == "object" {
		obj, ok := data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected object, got %T", data)
		}

		for _, required := range schema.Required {
			if _, exists := obj[required]; !exists {
				return fmt.Errorf("missing required field: %s", required)
			}
		}

		for name, prop := range schema.Properties {
			if value, exists := obj[name]; exists {
				if err := validateProperty(value, prop); err != nil {
					return fmt.Errorf("invalid field %s: %w", name, err)
				}
			}
		}
	}
	return nil
}

func validateProperty(value interface{}, prop types.Property) error {
	switch prop.Type {
	case "array":
		arr, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
		if prop.MinimumItems > 0 && len(arr) < prop.MinimumItems {
			return fmt.Errorf("array has too few items")
		}
		if prop.MaximumItems > 0 && len(arr) > prop.MaximumItems {
			return fmt.Errorf("array has too many items")
		}

	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
		if len(prop.Enum) > 0 {
			valid := false
			for _, enum := range prop.Enum {
				if str == enum {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid enum value")
			}
		}
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

const payloadFixPromptTemplate = `Your previous JSON payload had validation errors:

Previous Payload:
{{.previousPayload}}

Validation Errors:
{{.errors}}

Please provide a new JSON payload that fixes these validation errors. The payload must conform to the original schema:

Schema Details:
{{.schemaStr}}

Task Context:
{{- if .taskContext}}
{{.taskContext}}
{{- end}}

IMPORTANT:
- Return ONLY a valid JSON object with ALL required fields
- Ensure all nested objects have their required fields
- Fix ALL validation errors mentioned above
- Keep any valid parts from the previous payload

Response format: JSON object only.`

// Add these methods to PayloadAgent struct
func (a *PayloadAgent) GeneratePayloadWithRetry(ctx context.Context, task *types.Task, action types.Action) ([]byte, error) {
	payload, err := a.GeneratePayload(ctx, task, action)
	if err != nil {
		return nil, err
	}

	// Try to send the payload to the endpoint (we'll need the URL and method from the action)
	client := &http.Client{Timeout: 30 * time.Second}
	url := fmt.Sprintf("%s%s", action.Name, action.Path)

	req, err := http.NewRequestWithContext(ctx, action.Method, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// If we get a 422 (Validation Error), try to fix the payload
	if resp.StatusCode == http.StatusUnprocessableEntity {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading error response: %w", err)
		}

		// Try to fix the payload with the validation errors
		return a.fixPayloadWithErrors(ctx, task, action, payload, body)
	}

	return payload, nil
}

func (a *PayloadAgent) fixPayloadWithErrors(ctx context.Context, task *types.Task, action types.Action, previousPayload []byte, errorBody []byte) ([]byte, error) {
	var validationResp struct {
		Detail []struct {
			Loc  []interface{} `json:"loc"`
			Msg  string        `json:"msg"`
			Type string        `json:"type"`
		} `json:"detail"`
	}

	if err := json.Unmarshal(errorBody, &validationResp); err != nil {
		return nil, fmt.Errorf("parsing validation errors: %w", err)
	}

	// Format validation errors for the prompt
	var errorsStr strings.Builder
	for _, detail := range validationResp.Detail {
		path := make([]string, 0)
		for _, loc := range detail.Loc[1:] { // Skip "body" from the path
			path = append(path, fmt.Sprint(loc))
		}
		errorsStr.WriteString(fmt.Sprintf("- %s at path: %s\n", detail.Msg, strings.Join(path, ".")))
	}

	// Prepare prompt data for fixing the payload
	promptData := map[string]interface{}{
		"previousPayload": string(previousPayload),
		"errors":          errorsStr.String(),
		"schemaStr":       formatSchema(action.InputSchema),
		"taskContext":     generateTaskContext(task),
	}

	// Generate the fix prompt
	prompt, err := a.promptMgr.GeneratePrompt("payloadFixPrompt", promptData)
	if err != nil {
		return nil, fmt.Errorf("generating fix prompt: %w", err)
	}

	// Get completion for the fix
	completion, err := getCompletionWithRetries(ctx, a.llmClient, prompt)
	if err != nil {
		return nil, err
	}

	// Validate and format the fixed JSON
	return validateAndFormatJSON(completion, action.InputSchema)
}
