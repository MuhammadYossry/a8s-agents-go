package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Executor interface {
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
}

type TaskExecutor struct {
	AgentDef *AgentDefinition `json:"agentDefination"`
}

func NewTaskExecutor(agentDefinition *AgentDefinition) *TaskExecutor {
	return &TaskExecutor{AgentDef: agentDefinition}
}

func (e *TaskExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	// Simulate some work being don
	if e.AgentDef.Type == "external" {
		e.externalExecute(ctx, task)
	}

	return &TaskResult{
		TaskID:     task.ID,
		Success:    true,
		FinishedAt: time.Now(),
	}, nil
}

// CodeRequirements represents the structure for code generation request
type CodeRequirements struct {
	Description         string   `json:"description"`
	RequiredFunctions   []string `json:"required_functions"`
	Dependencies        []string `json:"dependencies"`
	PythonVersion       string   `json:"python_version"`
	TestingRequirements []string `json:"testing_requirements"`
}

// GenerateCodeRequest represents the complete request body
type GenerateCodeRequest struct {
	CodeRequirements   CodeRequirements `json:"code_requirements"`
	StyleGuide         string           `json:"style_guide"`
	IncludeTests       bool             `json:"include_tests"`
	DocumentationLevel string           `json:"documentation_level"`
}

// GenerateCodeResponse represents the expected response structure based on OutputSchema
type GenerateCodeResponse struct {
	GeneratedCode string `json:"generatedCode"`
	Description   string `json:"description"`
	TestCases     string `json:"testCases"`
	Documentation string `json:"documentation"`
}

type SchemaParser struct {
	inputSchema  SchemaConfig
	outputSchema SchemaConfig
}

func NewSchemaParser(action Action) *SchemaParser {
	return &SchemaParser{
		inputSchema:  action.InputSchema,
		outputSchema: action.OutputSchema,
	}
}

func (p *SchemaParser) ValidateAndPrepareRequest(data map[string]interface{}) error {
	log.Printf("Validating fields: %v against required: %v", getKeys(data), p.inputSchema.Required)
	for _, field := range p.inputSchema.Required {
		if _, exists := data[field]; !exists {
			return fmt.Errorf("missing required field: %s (available fields: %v)",
				field, getKeys(data))
		}
	}
	return nil
}

func (p *SchemaParser) ValidateResponse(response map[string]interface{}) error {
	for _, field := range p.outputSchema.Required {
		if _, exists := response[field]; !exists {
			return fmt.Errorf("response missing required field: %s", field)
		}
	}
	return nil
}

func (e *TaskExecutor) externalExecute(ctx context.Context, task *Task) (*TaskResult, error) {
	// Find matching action
	var targetAction Action
	found := false
	for _, action := range e.AgentDef.Actions {
		for taskType, skills := range e.AgentDef.SkillsByType {
			if task.Type == taskType {
				for _, skill := range skills {
					if skill == task.SkillsRequired[0] && action.Name == skill {
						targetAction = action
						found = true
						break
					}
				}
			}
		}
		if found {
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("no matching action found for task type %s and skill %s",
			task.Type, task.SkillsRequired[0])
	}

	// Create schema parser
	parser := NewSchemaParser(targetAction)

	// Prepare request body
	reqBody := map[string]interface{}{
		"codeRequirements": map[string]interface{}{ // Changed from code_requirements to codeRequirements
			"description":          "Create a REST API endpoint",
			"required_functions":   []string{"get_user", "create_user"},
			"dependencies":         []string{"fastapi", "sqlalchemy"},
			"python_version":       "3.9",
			"testing_requirements": []string{"pytest"},
		},
		"styleGuide":         "PEP8",     // Changed from style_guide to styleGuide
		"includeTests":       true,       // Changed from include_tests to includeTests
		"documentationLevel": "detailed", // Changed from documentation_level to documentationLevel
	}

	// Log the schema requirements for debugging
	log.Printf("Input Schema Required Fields: %v", targetAction.InputSchema.Required)
	log.Printf("Request Body Keys: %v", getKeys(reqBody))

	// Validate request against schema
	if err := parser.ValidateAndPrepareRequest(reqBody); err != nil {
		log.Printf("Validation Error: %v", err)
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Marshal request body
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Construct URL and create request
	url := fmt.Sprintf("%s%s", e.AgentDef.BaseURL, targetAction.Path)
	req, err := http.NewRequestWithContext(ctx, targetAction.Method, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Agent request creation error: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Agent request execution error: %v", err)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		var response map[string]interface{}
		log.Print(json.Unmarshal(body, &response))
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse and validate response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if err := parser.ValidateResponse(response); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	log.Printf("Successfully executed task %s with response: %+v", task.ID, response)

	return &TaskResult{
		TaskID:     task.ID,
		Output:     body,
		Success:    true,
		FinishedAt: time.Now(),
	}, nil
}

// Helper function to get map keys for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
