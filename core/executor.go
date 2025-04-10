// executor.go
package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	url2 "net/url"
	"strings"
	"time"

	"github.com/MuhammadYossry/a8s-agents-go/types"
)

type TaskExecutor struct {
	AgentDef     *types.AgentDefinition
	payloadAgent types.PayloadAgent
	client       *http.Client
}

func NewTaskExecutor(config types.TaskExecutorConfig) *TaskExecutor {
	if config.HTTPTimeout == 0 {
		config.HTTPTimeout = 30 * time.Second
	}

	return &TaskExecutor{
		AgentDef:     config.AgentDefinition,
		payloadAgent: config.PayloadAgent,
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

func validateMatchResult(result *types.MatchResult) error {
	if result == nil {
		return fmt.Errorf("match result is nil")
	}

	if result.Matched && result.Match == nil {
		return fmt.Errorf("matched is true but match details are missing")
	}

	if result.Matched {
		if result.Match.AgentID == "" {
			return fmt.Errorf("agent ID is required for matched result")
		}
		if result.Match.Action == "" {
			return fmt.Errorf("action is required for matched result")
		}
		if result.Match.Confidence < 0 || result.Match.Confidence > 100 {
			return fmt.Errorf("confidence must be between 0 and 100")
		}
	}

	return nil
}

func convertMatchResultToActionPlan(result *types.MatchResult) *types.ActionPlan {
	if !result.Matched || result.Match == nil {
		return &types.ActionPlan{
			SelectedAction: "",
			Confidence:     0,
			Reasoning: types.ActionPlanReasoning{
				PrimaryReason: result.Error,
			},
			Validation: types.ActionValidation{
				FrameworkCompatible: false,
				SkillPathSupported:  false,
				MissingRequirements: []string{"No matching agent found"},
			},
		}
	}

	return &types.ActionPlan{
		SelectedAction: result.Match.Action,
		Confidence:     result.Match.Confidence / 100.0, // Convert 0-100 to 0-1 scale
		Reasoning: types.ActionPlanReasoning{
			PrimaryReason: result.Match.Reasoning,
			AlignmentPoints: []string{
				fmt.Sprintf("Agent %s selected", result.Match.AgentID),
				fmt.Sprintf("Path match score: %.2f", result.Match.MatchDetails.PathMatchScore),
				fmt.Sprintf("Framework score: %.2f", result.Match.MatchDetails.FrameworkScore),
				fmt.Sprintf("Action score: %.2f", result.Match.MatchDetails.ActionScore),
				fmt.Sprintf("Version score: %.2f", result.Match.MatchDetails.VersionScore),
			},
		},
		Implementation: types.ActionImplementation{
			RequiredParameters: make(map[string]interface{}),
		},
		Validation: types.ActionValidation{
			FrameworkCompatible: result.Match.MatchDetails.FrameworkScore >= 10,
			SkillPathSupported:  result.Match.MatchDetails.PathMatchScore >= 20,
			MissingRequirements: []string{},
		},
	}
}
func (e *TaskExecutor) Execute(ctx context.Context, task *types.Task) (*types.TaskResult, error) {
	// Handle internal agents differently
	// if e.AgentDef.Type != "external" {
	// 	// For internal agents, we might want to implement different logic
	// 	// For now, just return success
	// 	return &types.TaskResult{
	// 		TaskID:     task.ID,
	// 		Success:    true,
	// 		FinishedAt: time.Now(),
	// 	}, nil
	// }

	// Find appropriate action
	// _, err := e.findActionForTask(ctx, task)
	// if err != nil {
	return &types.TaskResult{
		TaskID:     task.ID,
		Success:    false,
		Error:      fmt.Sprintf("action execution wip failed W"),
		FinishedAt: time.Now(),
	}, nil
	// }

	// Execute the action with proper error handling
	// return e.executeAction(ctx, task, *action)
}

func (e *TaskExecutor) executeAction(ctx context.Context, task *types.Task, action types.Action) (*types.TaskResult, error) {
	baseURL := strings.TrimRight(e.AgentDef.BaseURL, "/")
	if baseURL == "localhost" || strings.HasPrefix(baseURL, "localhost:") {
		baseURL = "http://" + baseURL
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	// Ensure path starts with a single slash
	path := "/" + strings.TrimLeft(action.Path, "/")

	url := baseURL + path
	log.Printf("Constructed URL for task %s: %s", task.ID, url)

	// Create a copy of the action with the properly formatted URL
	actionCopy := action
	actionCopy.BaseURL = baseURL

	// Prepare request with proper error handling
	req, err := e.prepareRequest(ctx, task, actionCopy, url)
	if err != nil {
		log.Printf("Error preparing request for task %s: %v", task.ID, err)
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      fmt.Sprintf("request preparation failed: %v", err),
			FinishedAt: time.Now(),
		}, nil
	}

	// Log request details
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			log.Printf("Error reading request body for task %s: %v", task.ID, err)
		} else {
			// Restore the body for the actual request
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			log.Printf("Request payload for task %s: %s", task.ID, string(bodyBytes))
		}
	} else {
		log.Printf("Warning: Request body is nil for task %s", task.ID)
	}

	// Execute request with retries
	var resp *http.Response
	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d for task %s", attempt+1, maxRetries, task.ID)
			select {
			case <-ctx.Done():
				log.Printf("Task %s cancelled during retry attempt %d", task.ID, attempt+1)
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
		log.Printf("Request attempt %d failed for task %s: %v", attempt+1, task.ID, err)
	}

	if err != nil {
		log.Printf("All retry attempts failed for task %s: %v", task.ID, lastErr)
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      fmt.Sprintf("request failed after %d attempts: %v", maxRetries, lastErr),
			FinishedAt: time.Now(),
		}, nil
	}

	defer resp.Body.Close()
	log.Printf("Received response with status %d for task %s", resp.StatusCode, task.ID)

	// Handle response
	result, err := e.handleResponse(resp, task)
	if err != nil {
		log.Printf("Error handling response for task %s: %v", task.ID, err)
		return &types.TaskResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      fmt.Sprintf("response handling failed: %v", err),
			FinishedAt: time.Now(),
		}, nil
	}

	if !result.Success {
		log.Printf("Task %s completed but unsuccessful: %s", task.ID, result.Error)
	} else {
		log.Printf("Successfully executed task %s", task.ID)
	}

	result.FinishedAt = time.Now()
	return result, nil
}

func (e *TaskExecutor) prepareRequest(ctx context.Context, task *types.Task, action types.Action, url string) (*http.Request, error) {
	// Validate URL before proceeding
	if _, err := url2.Parse(url); err != nil {
		return nil, fmt.Errorf("invalid URL %s: %v", url, err)
	}

	var payload []byte
	var err error

	// Generate or use existing payload
	if len(task.Payload) == 0 {
		if e.payloadAgent == nil {
			return nil, fmt.Errorf("no payload provided and no payload agent available")
		}

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
		log.Printf("Error reading response body for task %s: %v", task.ID, err)
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// Handle empty response
	if len(body) == 0 {
		log.Printf("Empty response body received for task %s", task.ID)
		return nil, fmt.Errorf("empty response body")
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		errorMsg := fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body))
		log.Printf("Request failed for task %s: %s", task.ID, errorMsg)
		return &types.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   errorMsg,
		}, nil
	}

	// Validate response format
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("Invalid JSON response for task %s: %v", task.ID, err)
		return nil, fmt.Errorf("invalid response JSON: %w", err)
	}

	log.Printf("Successfully completed task %s with status %d", task.ID, resp.StatusCode)
	return &types.TaskResult{
		TaskID:  task.ID,
		Success: true,
		Output:  body,
	}, nil
}
