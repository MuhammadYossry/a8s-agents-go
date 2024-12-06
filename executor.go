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
		log.Print(e.AgentDef)
		// the statment output &{python-code-agent external Advanced Python code generation, testing, and deployment agent http://localhost:9200/v1 [pythonCodeTask pythonTestingTask pythonDeploymentTask] map[pythonCodeTask:[generateCode improveCode reviewCode formatCode] pythonDeploymentTask:[deployPreview configureEnvironment manageSecrets] pythonTestingTask:[generateTests runTests analyzeCoverage]] [{generateCode /code_agent/python/generate_code POST {json [codeRequirements] [] map[]} {json [] [generatedCode description testCases documentation] map[]}} {improveCode /code_agent/python/improve_code POST {json [changesList] [] map[]} {json [] [codeChanges changesDescription qualityMetrics] map[]}} {testCode /code_agent/python/test_code POST {json [testType requirePassing testInstructions codeToTest] [] map[]} {json [] [codeTests testsDescription coverageStatus] map[]}} {deployPreview /deploy_agent/python/preview POST {json [branchID isPrivate] [] map[]} {json [] [previewURL isPrivate HTTPAuth deploymentTime] map[]}}]}
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

func (e *TaskExecutor) externalExecute(ctx context.Context, task *Task) (*TaskResult, error) {
	if task.Type == "pythonCodeTask" && task.SkillsRequired[0] == "generateCode" {
		// Find the matching action from AgentDef
		var targetAction Action
		for _, action := range e.AgentDef.Actions {
			if action.Name == "generateCode" {
				targetAction = action
				break
			}
		}

		// Construct the full URL
		url := fmt.Sprintf("%s%s", e.AgentDef.BaseURL, targetAction.Path)

		// Create the request payload
		reqBody := GenerateCodeRequest{
			CodeRequirements: CodeRequirements{
				Description:         "Create a REST API endpoint",
				RequiredFunctions:   []string{"get_user", "create_user"},
				Dependencies:        []string{"fastapi", "sqlalchemy"},
				PythonVersion:       "3.9",
				TestingRequirements: []string{"pytest"},
			},
			StyleGuide:         "PEP8",
			IncludeTests:       true,
			DocumentationLevel: "detailed",
		}

		// Marshal the request body
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		// Create the HTTP request
		req, err := http.NewRequestWithContext(ctx, targetAction.Method, url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")

		// Make the request
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}
		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		// Check status code
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
		}

		// Parse response into a dynamic map
		var response map[string]interface{}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// Log the raw response
		log.Printf("Raw Response Body: %s", string(body))

		// Log the parsed response
		log.Printf("Parsed Response: %+v", response)

		return &TaskResult{
			TaskID:     task.ID,
			Success:    true,
			FinishedAt: time.Now(),
		}, nil
	}

	return &TaskResult{
		TaskID:     task.ID,
		Success:    true,
		FinishedAt: time.Now(),
	}, nil
}
