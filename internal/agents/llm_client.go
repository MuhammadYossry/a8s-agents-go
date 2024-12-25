// llm_client.go
package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LLMProvider string

const (
	OpenAI LLMProvider = "openai"
	Qwen   LLMProvider = "qwen"
)

const DefaultSystemMessage = "You are a helpful assistant."

type LLMConfig struct {
	Provider      LLMProvider
	BaseURL       string
	APIKey        string
	Model         string
	Timeout       time.Duration
	SystemMessage string
	Options       map[string]interface{} // Additional provider-specific options
}

type LLMClient struct {
	config     *LLMConfig
	httpClient *http.Client
}

func NewLLMClient(config *LLMConfig) *LLMClient {
	if config.Options == nil {
		config.Options = make(map[string]interface{})
	}

	// Set default options if not provided
	if _, ok := config.Options["temperature"]; !ok {
		config.Options["temperature"] = 0.7
	}

	// Set default system message if not provided
	if config.SystemMessage == "" {
		config.SystemMessage = DefaultSystemMessage
	}

	return &LLMClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CreateChatCompletionWithSystemMessage allows overriding the system message for a specific request
func (c *LLMClient) CreateChatCompletionWithSystemMessage(ctx context.Context, prompt, systemMessage string) (string, error) {
	if systemMessage == "" {
		systemMessage = c.config.SystemMessage
	}
	return c.createCompletion(ctx, prompt, systemMessage)
}

// CreateChatCompletion uses the default system message
func (c *LLMClient) CreateChatCompletion(ctx context.Context, prompt string) (string, error) {
	return c.createCompletion(ctx, prompt, c.config.SystemMessage)
}

func (c *LLMClient) createCompletion(ctx context.Context, prompt, systemMessage string) (string, error) {
	var requestBody []byte
	var err error

	switch c.config.Provider {
	case OpenAI:
		requestBody, err = c.createOpenAIRequest(prompt, systemMessage)
	case Qwen:
		requestBody, err = c.createQwenRequest(prompt, systemMessage)
	default:
		return "", fmt.Errorf("unsupported LLM provider: %s", c.config.Provider)
	}

	if err != nil {
		return "", fmt.Errorf("creating request body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.config.BaseURL+"/chat/completions",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return c.parseResponse(body)
}

func (c *LLMClient) createOpenAIRequest(prompt, systemMessage string) ([]byte, error) {
	payload := map[string]interface{}{
		"model": c.config.Model,
		"messages": []Message{
			{
				Role:    "system",
				Content: systemMessage,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		"temperature": c.config.Options["temperature"],
	}

	return json.Marshal(payload)
}

func (c *LLMClient) createQwenRequest(prompt, systemMessage string) ([]byte, error) {
	payload := map[string]interface{}{
		"model": c.config.Model,
		"messages": []Message{
			{
				Role:    "system",
				Content: systemMessage,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		"temperature":   c.config.Options["temperature"],
		"top_p":         0.8,
		"result_format": "message",
		"stream":        false,
	}

	for k, v := range c.config.Options {
		if k != "temperature" {
			payload[k] = v
		}
	}

	return json.Marshal(payload)
}

func (c *LLMClient) parseResponse(body []byte) (string, error) {
	switch c.config.Provider {
	case OpenAI:
		var result struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return "", fmt.Errorf("parsing OpenAI response: %w", err)
		}
		if len(result.Choices) == 0 {
			return "", fmt.Errorf("no completion choices returned")
		}
		return result.Choices[0].Message.Content, nil

	case Qwen:
		var result struct {
			Output struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			} `json:"output"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return "", fmt.Errorf("parsing Qwen response: %w", err)
		}
		if len(result.Output.Choices) == 0 {
			return "", fmt.Errorf("no completion choices returned")
		}
		return result.Output.Choices[0].Message.Content, nil

	default:
		return "", fmt.Errorf("unsupported LLM provider: %s", c.config.Provider)
	}
}
