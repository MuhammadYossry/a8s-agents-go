// executor.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Executor interface {
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
}

type TaskExecutor struct {
	AgentDef *AgentDefinition
}

func NewTaskExecutor(agentDefinition *AgentDefinition) *TaskExecutor {
	return &TaskExecutor{AgentDef: agentDefinition}
}

func (e *TaskExecutor) findActionForTask(task *Task) (*Action, error) {
	matcher := NewCapabilityMatcher(nil, DefaultMatcherConfig())
	matches := matcher.calculateAgentMatch(task.Requirements, AgentCapability{
		Capabilities: e.AgentDef.Capabilities,
		Actions:      e.AgentDef.Actions,
	})

	if matches.Score < DefaultMatcherConfig().MinimumScore {
		return nil, fmt.Errorf("no suitable action found for task requirements")
	}

	actionCopy := matches.Action
	return &actionCopy, nil
}

func (e *TaskExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	if e.AgentDef.Type != "external" {
		return &TaskResult{
			TaskID:     task.ID,
			Success:    true,
			FinishedAt: time.Now(),
		}, nil
	}

	action, err := e.findActionForTask(task)
	if err != nil {
		return nil, err
	}

	return e.executeAction(ctx, task, *action)
}

func (e *TaskExecutor) executeAction(ctx context.Context, task *Task, action Action) (*TaskResult, error) {
	url := fmt.Sprintf("%s%s", e.AgentDef.BaseURL, action.Path)

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: true,
		},
	}

	req, err := e.prepareRequest(ctx, task, action, url)
	if err != nil {
		return nil, fmt.Errorf("preparing request: %w", err)
	}
	fmt.Printf("Request_body: %v", req.Body)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	return e.handleResponse(resp, task)
}

func (e *TaskExecutor) prepareRequest(ctx context.Context, task *Task, action Action, url string) (*http.Request, error) {
	reqBody := make(map[string]interface{})
	if err := json.Unmarshal(task.Payload, &reqBody); err != nil {
		return nil, fmt.Errorf("parsing payload: %w", err)
	}

	parser := NewSchemaParser(action)
	if err := parser.ValidateAndPrepareRequest(reqBody); err != nil {
		return nil, fmt.Errorf("validating request: %w", err)
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Print for debugging
	fmt.Printf("Request payload: %s\n", string(jsonBody))

	return http.NewRequestWithContext(ctx, action.Method, url, bytes.NewBuffer(jsonBody))
}

func (e *TaskExecutor) handleResponse(resp *http.Response, task *Task) (*TaskResult, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("error response body: %s \n", body)
		return &TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)),
			FinishedAt: time.Now(),
		}, nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	fmt.Printf("agent response: %s", body)

	return &TaskResult{
		TaskID:     task.ID,
		Success:    true,
		Output:     body,
		FinishedAt: time.Now(),
	}, nil
}
