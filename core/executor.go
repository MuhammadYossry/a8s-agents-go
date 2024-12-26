// executor.go
package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/internal/agents"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

type Executor interface {
	Execute(ctx context.Context, task *types.Task) (*types.TaskResult, error)
}

type TaskExecutor struct {
	AgentDef           *AgentDefinition
	payloadAgent       *agents.PayloadAgent
	actionPlannerAgent *agents.ActionPlannerAgent
	client             *http.Client
}

type TaskExecutorConfig struct {
	AgentDefinition    *AgentDefinition
	PayloadAgent       *agents.PayloadAgent
	ActionPlannerAgent *agents.ActionPlannerAgent
	HTTPTimeout        time.Duration
}

func NewTaskExecutor(config TaskExecutorConfig) *TaskExecutor {
	if config.HTTPTimeout == 0 {
		config.HTTPTimeout = 30 * time.Second
	}

	return &TaskExecutor{
		AgentDef:           config.AgentDefinition,
		payloadAgent:       config.PayloadAgent,
		actionPlannerAgent: config.ActionPlannerAgent,
		client: &http.Client{
			Timeout: config.HTTPTimeout,
			Transport: &http.Transport{
				MaxIdleConns:       100,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: true,
			},
		},
	}
}

// func (e *TaskExecutor) findActionForTask(task *types.Task) (*types.Action, error) {
// 	matcher := NewCapabilityMatcher(nil, DefaultMatcherConfig())
// 	matches := matcher.calculateAgentMatch(task.Requirements, types.AgentCapability{
// 		Capabilities: e.AgentDef.Capabilities,
// 		Actions:      e.AgentDef.Actions,
// 	})

// 	if matches.Score < DefaultMatcherConfig().MinimumScore {
// 		return nil, fmt.Errorf("no suitable action found for task requirements")
// 	}

// 	actionCopy := matches.Action
// 	return &actionCopy, nil
// }

func (e *TaskExecutor) findActionForTask(ctx context.Context, task *types.Task) (*types.Action, error) {
	// If no action planner agent is available, fall back to capability matcher
	if e.actionPlannerAgent == nil {
		matcher := NewCapabilityMatcher(nil, DefaultMatcherConfig())
		matches := matcher.calculateAgentMatch(task.Requirements, types.AgentCapability{
			Capabilities: e.AgentDef.Capabilities,
			Actions:      e.AgentDef.Actions,
		})

		if matches.Score < DefaultMatcherConfig().MinimumScore {
			return nil, fmt.Errorf("no suitable action found for task requirements")
		}

		actionCopy := matches.Action
		return &actionCopy, nil
	}

	// Use ActionPlannerAgent to determine the most suitable action
	actionPlan, err := e.actionPlannerAgent.PlanAction(ctx, task, e.AgentDef.Actions)
	if err != nil {
		return nil, fmt.Errorf("planning action: %w", err)
	}

	// Validate action plan confidence
	if actionPlan.Confidence < 0.7 { // You might want to make this threshold configurable
		return nil, fmt.Errorf("low confidence (%f) in action selection", actionPlan.Confidence)
	}

	// Find the selected action in the agent's available actions
	var selectedAction *types.Action
	for _, action := range e.AgentDef.Actions {
		if action.Name == actionPlan.SelectedAction {
			selectedAction = &action
			break
		}
	}

	if selectedAction == nil {
		return nil, fmt.Errorf("selected action '%s' not found in available actions", actionPlan.SelectedAction)
	}

	// Validate framework compatibility if specified in task requirements
	if framework, ok := task.Requirements.Parameters["framework"].(string); ok {
		if !actionPlan.Validation.FrameworkCompatible {
			return nil, fmt.Errorf("selected action '%s' is not compatible with framework '%s'",
				actionPlan.SelectedAction, framework)
		}
	}

	// Validate skill path support
	if !actionPlan.Validation.SkillPathSupported {
		return nil, fmt.Errorf("selected action '%s' does not support required skill path %v",
			actionPlan.SelectedAction, task.Requirements.SkillPath)
	}

	// Check for missing requirements
	if len(actionPlan.Validation.MissingRequirements) > 0 {
		return nil, fmt.Errorf("missing requirements for action '%s': %v",
			actionPlan.SelectedAction, actionPlan.Validation.MissingRequirements)
	}

	return selectedAction, nil
}

func (e *TaskExecutor) Execute(ctx context.Context, task *types.Task) (*types.TaskResult, error) {
	// Handle internal agents differently
	if e.AgentDef.Type != "external" {
		// For internal agents, we might want to implement different logic
		// For now, just return success
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    true,
			FinishedAt: time.Now(),
		}, nil
	}

	// Find appropriate action
	action, err := e.findActionForTask(ctx, task)
	if err != nil {
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      fmt.Sprintf("action selection failed: %v", err),
			FinishedAt: time.Now(),
		}, nil
	}

	// Execute the action with proper error handling
	return e.executeAction(ctx, task, *action)
}

func (e *TaskExecutor) executeAction(ctx context.Context, task *types.Task, action types.Action) (*types.TaskResult, error) {
	url := fmt.Sprintf("%s%s", e.AgentDef.BaseURL, action.Path)

	// Prepare request with proper error handling
	req, err := e.prepareRequest(ctx, task, action, url)
	if err != nil {
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      fmt.Sprintf("request preparation failed: %v", err),
			FinishedAt: time.Now(),
		}, nil
	}

	// Execute request with retries
	var resp *http.Response
	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return &types.TaskResult{
					TaskID:     task.ID,
					Success:    false,
					Error:      "task cancelled during retry",
					FinishedAt: time.Now(),
				}, nil
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
				// Exponential backoff
			}
		}

		resp, err = e.client.Do(req)
		if err == nil {
			break
		}
		lastErr = err
	}

	if err != nil {
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      fmt.Sprintf("request failed after %d attempts: %v", maxRetries, lastErr),
			FinishedAt: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	// Handle response
	result, err := e.handleResponse(resp, task)
	if err != nil {
		return &types.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   fmt.Sprintf("response handling failed: %v", err),
			// StartedAt:  startTime,
			FinishedAt: time.Now(),
		}, nil
	}

	// Add timing information
	// result.StartedAt = startTime
	result.FinishedAt = time.Now()

	return result, nil
}

func (e *TaskExecutor) prepareRequest(ctx context.Context, task *types.Task, action types.Action, url string) (*http.Request, error) {
	var payload []byte
	var err error

	// Generate or use existing payload
	if len(task.Payload) == 0 {
		if e.payloadAgent == nil {
			return nil, fmt.Errorf("no payload provided and no payload agent available")
		}

		// Use GeneratePayloadWithRetry instead of GeneratePayload
		payload, err = e.payloadAgent.GeneratePayloadWithRetry(ctx, task, action)
		if err != nil {
			return nil, fmt.Errorf("generating payload: %w", err)
		}

		if len(payload) == 0 {
			return nil, fmt.Errorf("payload agent generated empty payload")
		}
	} else {
		payload = task.Payload
	}

	// Validate payload
	var reqBody map[string]interface{}
	if err := json.Unmarshal(payload, &reqBody); err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}

	// Validate against schema
	parser := NewSchemaParser(action)
	if err := parser.ValidateAndPrepareRequest(reqBody); err != nil {
		return nil, fmt.Errorf("payload validation failed: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, action.Method, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (e *TaskExecutor) handleResponse(resp *http.Response, task *types.Task) (*types.TaskResult, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// Handle empty response
	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		return &types.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)),
		}, nil
	}

	// Validate response format
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("invalid response JSON: %w", err)
	}

	return &types.TaskResult{
		TaskID:  task.ID,
		Success: true,
		Output:  body,
	}, nil
}
