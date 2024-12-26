// llm_client.go
package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

// Debug mode flag
var Debug = true

type LLMProvider string

const (
	OpenAI LLMProvider = "openai"
	Qwen   LLMProvider = "qwen"
)

type LLMConfig struct {
	Provider      LLMProvider
	BaseURL       string
	APIKey        string
	Model         string
	Timeout       time.Duration
	SystemMessage string
	Options       map[string]interface{}
	Debug         bool
}

type LLMClient struct {
	config     *LLMConfig
	httpClient *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewLLMClient(config *LLMConfig) *LLMClient {
	if config == nil {
		panic("LLMConfig cannot be nil")
	}

	// Validate and sanitize BaseURL
	if config.BaseURL == "" {
		panic("BaseURL must be specified")
	}
	// Ensure BaseURL ends with "/"
	if config.BaseURL[len(config.BaseURL)-1] != '/' {
		config.BaseURL += "/"
	}

	if config.APIKey == "" {
		panic("APIKey must be specified")
	}
	if config.Model == "" {
		panic("Model must be specified")
	}

	// Set default timeout
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second // Increased timeout for large models
	}

	// Initialize default options
	if config.Options == nil {
		config.Options = make(map[string]interface{})
	}
	setDefaultOptions(config)

	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &loggingRoundTripper{
			next:    http.DefaultTransport,
			debug:   config.Debug,
			baseURL: config.BaseURL,
		},
	}

	return &LLMClient{
		config:     config,
		httpClient: client,
	}
}

func setDefaultOptions(config *LLMConfig) {
	defaults := map[string]interface{}{
		"temperature":   0.7,
		"top_p":         0.8,
		"result_format": "message",
		"stream":        false,
	}

	for k, v := range defaults {
		if _, exists := config.Options[k]; !exists {
			config.Options[k] = v
		}
	}
}

// Custom transport for logging requests and responses
type loggingRoundTripper struct {
	next    http.RoundTripper
	debug   bool
	baseURL string
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if l.debug {
		// Log request
		reqDump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Printf("Request to %s:\n%s\n", req.URL, string(reqDump))
		}
	}

	resp, err := l.next.RoundTrip(req)

	if err != nil {
		log.Printf("HTTP request error: %v\n", err)
		return nil, err
	}

	if l.debug && resp != nil {
		// Log response
		respDump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Printf("Response from %s:\n%s\n", req.URL, string(respDump))
		}
	}

	return resp, err
}

func (c *LLMClient) createCompletion(ctx context.Context, prompt, systemMessage string) (string, error) {
	// Ensure system message is not empty
	if systemMessage == "" {
		systemMessage = "You are a helpful AI assistant that generates valid JSON payloads based on the provided schema."
	}

	var requestBody []byte
	var err error

	requestBody, err = c.createQwenRequest(prompt, systemMessage)
	if err != nil {
		return "", fmt.Errorf("creating request body: %w", err)
	}

	endpoint := "chat/completions"
	if c.config.BaseURL[len(c.config.BaseURL)-1] != '/' {
		endpoint = "/" + endpoint
	}
	url := c.config.BaseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	if Debug {
		log.Printf("Making request to URL: %s\n", url)
		log.Printf("Request headers: %v\n", req.Header)
		log.Printf("Request body: %s\n", string(requestBody))
	}

	// Add retry logic for 502 errors
	maxRetries := 3
	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying request (attempt %d/%d) after error: %v", attempt+1, maxRetries, lastErr)
			time.Sleep(time.Duration(attempt) * 2 * time.Second) // Exponential backoff
		}

		resp, err = c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("sending request (attempt %d): %w", attempt+1, err)
			continue
		}

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("reading response body (attempt %d): %w", attempt+1, err)
			continue
		}

		if Debug {
			log.Printf("Response status: %d\n", resp.StatusCode)
			log.Printf("Response body: %s\n", string(body))
		}

		// Handle different status codes
		switch resp.StatusCode {
		case http.StatusOK:
			return c.parseQwenResponse(body)
		case http.StatusBadGateway: // 502
			lastErr = fmt.Errorf("backend server error (status 502)")
			continue
		case http.StatusServiceUnavailable: // 503
			lastErr = fmt.Errorf("service unavailable (status 503)")
			continue
		default:
			if len(body) == 0 {
				lastErr = fmt.Errorf("empty response with status code %d", resp.StatusCode)
			} else {
				lastErr = fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			}
			if resp.StatusCode < 500 { // Don't retry 4xx errors
				return "", lastErr
			}
			continue
		}
	}

	return "", fmt.Errorf("all retry attempts failed. Last error: %w", lastErr)
}

func (c *LLMClient) createQwenRequest(prompt, systemMessage string) ([]byte, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: systemMessage,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	request := map[string]interface{}{
		"model":    c.config.Model,
		"messages": messages,
	}

	// Add all options from config
	for k, v := range c.config.Options {
		request[k] = v
	}

	if Debug {
		log.Printf("Qwen request payload: %+v\n", request)
	}

	return json.Marshal(request)
}

func (c *LLMClient) parseQwenResponse(body []byte) (string, error) {
	// Parse the actual Qwen response structure
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		// Try parsing as error response
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if errErr := json.Unmarshal(body, &errorResp); errErr == nil && errorResp.Error.Message != "" {
			return "", fmt.Errorf("LLM API error: %s (type: %s, code: %s)",
				errorResp.Error.Message, errorResp.Error.Type, errorResp.Error.Code)
		}
		return "", fmt.Errorf("parsing Qwen response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	if result.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty content in response")
	}

	return result.Choices[0].Message.Content, nil
}

// CreateChatCompletion creates a chat completion with the default system message
func (c *LLMClient) CreateChatCompletion(ctx context.Context, prompt string) (string, error) {
	return c.createCompletion(ctx, prompt, c.config.SystemMessage)
}
