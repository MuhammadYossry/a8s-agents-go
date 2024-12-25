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
	if e.AgentDef.Type != "external" {
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    true,
			FinishedAt: time.Now(),
		}, nil
	}

	action, err := e.findActionForTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("finding action: %w", err)
	}

	return e.executeAction(ctx, task, *action)
}

func (e *TaskExecutor) executeAction(ctx context.Context, task *types.Task, action types.Action) (*types.TaskResult, error) {
	url := fmt.Sprintf("%s%s", e.AgentDef.BaseURL, action.Path)

	req, err := e.prepareRequest(ctx, task, action, url)
	if err != nil {
		return nil, fmt.Errorf("preparing request: %w", err)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	return e.handleResponse(resp, task)
}

func (e *TaskExecutor) prepareRequest(ctx context.Context, task *types.Task, action types.Action, url string) (*http.Request, error) {
	var payload []byte
	var err error
	fmt.Println(len(task.Payload))

	// If task payload is empty, generate it using PayloadAgent
	if len(task.Payload) == 0 && e.payloadAgent != nil {
		// Extract capabilities for context
		capabilities := make([]string, 0)
		for _, cap := range e.AgentDef.Capabilities {
			if desc, ok := cap.Metadata["description"].(string); ok {
				capabilities = append(capabilities, desc)
			}
		}

		// Generate payload using PayloadAgent
		payload, err = e.payloadAgent.GeneratePayload(ctx, task, action)
		if err != nil {
			return nil, fmt.Errorf("generating payload: %w", err)
		}
	} else {
		payload = task.Payload
	}

	// Parse and validate payload
	var reqBody map[string]interface{}
	if err := json.Unmarshal(payload, &reqBody); err != nil {
		return nil, fmt.Errorf("parsing payload: %w", err)
	}

	parser := NewSchemaParser(action)
	if err := parser.ValidateAndPrepareRequest(reqBody); err != nil {
		return nil, fmt.Errorf("validating request: %w", err)
	}

	// Marshal back to JSON
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, action.Method, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (e *TaskExecutor) handleResponse(resp *http.Response, task *types.Task) (*types.TaskResult, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)),
			FinishedAt: time.Now(),
		}, nil
	}

	// Parse response for validation
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &types.TaskResult{
		TaskID:     task.ID,
		Success:    true,
		Output:     body,
		FinishedAt: time.Now(),
	}, nil
}
