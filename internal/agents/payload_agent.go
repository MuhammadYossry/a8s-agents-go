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

const payloadPromptTemplate = `Create a payload matching this schema considering the task *context*:

Schema: {{.schemaStr}}

Type Details:
{{.typeDetails}}

Constraints:
{{.constraints}}

{{if .taskContext}}Context:
{{.taskContext}}{{end}}

Example:
{{.exampleStr}}

IMPORTANT:
- Return pure JSON only without markdown or code blocks
- Arrays must use [] even for single items (e.g. ["item"] not "item")
- Required fields: {{.requiredFields}}
- Follow all type constraints and patterns
- Respect enums and const values

Remember: Return ONLY the JSON object, without kind of quotes or single quotes or any other formatting.`

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

func (p *PayloadAgent) GeneratePayload(ctx context.Context, task *types.Task, action types.Action) ([]byte, error) {
	// Extract defs from schema
	defs := make(map[string]types.SchemaConfig)
	if action.InputSchema.Defs != nil {
		defs = action.InputSchema.Defs
	}

	promptData := map[string]interface{}{
		"schemaStr":      formatSchema(action.InputSchema),
		"typeDetails":    formatTypeDetails(action.InputSchema, defs),
		"constraints":    formatConstraints(action.InputSchema),
		"requiredFields": strings.Join(action.InputSchema.Required, ", "),
		"taskContext":    generateTaskContext(task),
		"exampleStr":     string(generateExample(action.InputSchema, task)),
	}

	prompt, err := p.promptMgr.GeneratePrompt("payloadPrompt", promptData)
	if err != nil {
		return nil, fmt.Errorf("generating prompt: %w", err)
	}

	completion, err := getCompletionWithRetries(ctx, p.llmClient, prompt)
	if err != nil {
		return nil, err
	}

	return validateAndFormatJSON(completion, action.InputSchema)
}

func formatSchema(schema types.SchemaConfig) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Type: %s\n", schema.Type))
	b.WriteString(fmt.Sprintf("Description: %s\n", schema.Description))
	return b.String()
}

func formatTypeDetails(schema types.SchemaConfig, defs map[string]types.SchemaConfig) string {
	var b strings.Builder

	for name, prop := range schema.Properties {

		b.WriteString(fmt.Sprintf("\n%s:\n", name))
		if prop.Ref != "" {
			if def, ok := resolveRef(prop.Ref, defs); ok {
				writeSchemaDetails(&b, def, "  ", defs)
			}
		} else if prop.AnyOf != nil {
			b.WriteString("  AnyOf:\n")
			for _, subProp := range prop.AnyOf {
				if subProp.Ref != "" {
					if def, ok := resolveRef(subProp.Ref, defs); ok {
						writeSchemaDetails(&b, def, "    ", defs)
					}
				} else {
					writePropertyDetails(&b, subProp, "    ", defs)
				}
			}
		} else {
			writePropertyDetails(&b, prop, "  ", defs)
		}
	}
	return b.String()
}

func writeSchemaDetails(b *strings.Builder, schema types.SchemaConfig, indent string, defs map[string]types.SchemaConfig) {
	b.WriteString(fmt.Sprintf("%sType: %s\n", indent, schema.Type))
	if schema.Description != "" {
		b.WriteString(fmt.Sprintf("%sDescription: %s\n", indent, schema.Description))
	}
	if len(schema.Properties) > 0 {
		b.WriteString(fmt.Sprintf("%sProperties:\n", indent))
		for name, prop := range schema.Properties {
			b.WriteString(fmt.Sprintf("%s  %s:\n", indent, name))
			writePropertyDetails(b, prop, indent+"    ", defs)
		}
	}
	if len(schema.Required) > 0 {
		b.WriteString(fmt.Sprintf("%sRequired: [%s]\n", indent, strings.Join(schema.Required, ", ")))
	}
}

func writePropertyDetails(b *strings.Builder, prop types.Property, indent string, defs map[string]types.SchemaConfig) {
	b.WriteString(fmt.Sprintf("%sType: %s\n", indent, prop.Type))

	if prop.Const != "" {
		b.WriteString(fmt.Sprintf("%sConst: %s\n", indent, prop.Const))
	}
	if len(prop.Enum) > 0 {
		b.WriteString(fmt.Sprintf("%sEnum: [%s]\n", indent, strings.Join(prop.Enum, ", ")))
	}
	if prop.Default != nil {
		b.WriteString(fmt.Sprintf("%sDefault: %v\n", indent, prop.Default))
	}
	if prop.Type == "array" && prop.Items != nil {
		b.WriteString(fmt.Sprintf("%sArray items:\n", indent))
		writePropertyDetails(b, *prop.Items, indent+"  ", defs)
	}
}

func resolveRef(ref string, defs map[string]types.SchemaConfig) (types.SchemaConfig, bool) {
	parts := strings.Split(strings.TrimPrefix(ref, "#/$defs/"), "/")
	if len(parts) > 0 {
		if def, ok := defs[parts[len(parts)-1]]; ok {
			return def, true
		}
	}
	return types.SchemaConfig{}, false
}

func formatValidationConstraints(b *strings.Builder, prop types.Property, indent string) {
	if prop.Pattern != "" {
		b.WriteString(fmt.Sprintf("%sPattern: %s\n", indent, prop.Pattern))
	}
	if prop.Minimum != nil {
		b.WriteString(fmt.Sprintf("%sMin: %v\n", indent, *prop.Minimum))
	}
	if prop.Maximum != nil {
		b.WriteString(fmt.Sprintf("%sMax: %v\n", indent, *prop.Maximum))
	}
	if prop.MinimumItems > 0 {
		b.WriteString(fmt.Sprintf("%sMin items: %d\n", indent, prop.MinimumItems))
	}
	if prop.MaximumItems > 0 {
		b.WriteString(fmt.Sprintf("%sMax items: %d\n", indent, prop.MaximumItems))
	}
}

func formatConstraints(schema types.SchemaConfig) string {
	var b strings.Builder
	formatSchemaConstraints(&b, schema, "", make(map[string]bool))
	return b.String()
}

func formatSchemaConstraints(b *strings.Builder, schema types.SchemaConfig, prefix string, visited map[string]bool) {
	for name, prop := range schema.Properties {
		fullName := prefix + name
		if visited[fullName] {
			continue
		}
		visited[fullName] = true

		if prop.Pattern != "" || prop.Minimum != nil || prop.Maximum != nil ||
			prop.MinimumItems > 0 || prop.MaximumItems > 0 {
			b.WriteString(fmt.Sprintf("\n%s:\n", fullName))
			formatValidationConstraints(b, prop, "  ")
		}

		// Recurse into nested objects
		if prop.Type == "object" && len(prop.Properties) > 0 {
			formatSchemaConstraints(b, types.SchemaConfig{Properties: prop.Properties}, fullName+".", visited)
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
	// Remove markdown code blocks and any surrounding whitespace
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "```json") {
		input = strings.TrimPrefix(input, "```json")
		input = strings.TrimSuffix(input, "```")
	} else if strings.HasPrefix(input, "```") {
		input = strings.TrimPrefix(input, "```")
		input = strings.TrimSuffix(input, "```")
	}
	input = strings.TrimSpace(input)

	var jsonData interface{}
	if err := json.Unmarshal([]byte(input), &jsonData); err != nil {
		// Add debug logging
		log.Printf("Failed to parse JSON input: %s", input)
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
// In payload_agent.go
func (a *PayloadAgent) GeneratePayloadWithRetry(ctx context.Context, task *types.Task, action types.Action) ([]byte, error) {
	var payload []byte
	var err error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d for generating payload for task %s", attempt+1, maxRetries, task.ID)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		// Generate initial payload
		payload, err = a.GeneratePayload(ctx, task, action)
		if err != nil {
			log.Printf("Error generating payload on attempt %d: %v", attempt+1, err)
			continue
		}

		// Validate payload against schema before sending
		var reqBody map[string]interface{}
		if err := json.Unmarshal(payload, &reqBody); err != nil {
			log.Printf("Invalid JSON payload on attempt %d: %v", attempt+1, err)
			continue
		}

		// Try to send the payload to the endpoint
		client := &http.Client{Timeout: 30 * time.Second}

		// Format URL properly
		baseURL := strings.TrimRight(action.BaseURL, "/")
		path := "/" + strings.TrimLeft(action.Path, "/")
		url := baseURL + path

		req, err := http.NewRequestWithContext(ctx, action.Method, url, bytes.NewBuffer(payload))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error sending request on attempt %d: %v", attempt+1, err)
			continue
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			log.Printf("Error reading response body on attempt %d: %v", attempt+1, err)
			continue
		}

		// Handle different response status codes
		switch resp.StatusCode {
		case http.StatusOK:
			return payload, nil
		case http.StatusUnprocessableEntity:
			log.Printf("Validation error on attempt %d, trying to fix payload", attempt+1)
			// Try to fix the payload with the validation errors
			if fixedPayload, err := a.fixPayloadWithErrors(ctx, task, action, payload, body); err == nil {
				return fixedPayload, nil
			}
		default:
			log.Printf("Unexpected status code %d on attempt %d", resp.StatusCode, attempt+1)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
	}

	return nil, fmt.Errorf("failed to generate valid payload after %d attempts", maxRetries)
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
