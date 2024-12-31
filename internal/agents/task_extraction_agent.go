// task_extraction_agent.go
package agents

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

// TaskExtractionAgent handles the analysis of user queries and converts them into structured task definitions
type TaskExtractionAgent struct {
	llmClient *LLMClient
	promptMgr *PromptManager
}

var (
	extractionInstance *TaskExtractionAgent
	extractionOnce     sync.Once
)

const taskExtractionPromptTemplate = `You are an expert system architect and task routing specialist. Your role is to analyze user queries and convert them into structured task definitions following strict output specifications.

CONTEXT:
- You must analyze technical queries and extract task-related information.
- The output must be valid JSON only, no explanations or additional text.
- *Task paths follow a hierarchical structure: Domain -> Subdomain -> Technology -> Action*.
- Common domains include: {{.domains}}

TASK REQUIREMENTS:
1. First, identify if the input is a task request.
2. Then, extract the core components: action description, technology, framework, and any specific requirements.
3. Finally, generate a structured JSON response.

THOUGHT PROCESS (internal only):
1. Is this a task request?
2. What is the primary actionDescription being requested?
3. What technology stack is involved?
4. What domain and subdomain does this belong to? Think if they match common domains first
5. Are there any specific parameters or requirements?

FORMAT SPECIFICATION:
{
    "title": "",      // Concise task title
    "description": "", // Detailed task description
    "requirements": {
        "skillPath": [], // Hierarchical path array: ["Development", "Backend"]
        "actionDescription": "",    // text to describe the action *role* description of the task needed to be performed
        "parameters": {} // Additional requirements parameters
    }
}

CONSTRAINTS:
- Output must be valid JSON only
- No explanatory text before or after JSON
- All fields must be present
- Skill paths must follow the hierarchical structure

SYSTEM NOTE:
Your task is to analyze the following query and output ONLY a JSON object following the above format.

USER QUERY:
{{.query}}`

// ExtractionResult represents the structured output of task extraction
type ExtractionResult struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Requirements struct {
		SkillPath         []string               `json:"skillPath"`
		ActionDescription string                 `json:"actionDescription"`
		Parameters        map[string]interface{} `json:"parameters"`
	} `json:"requirements"`
}

// GetTaskExtractionAgent returns a singleton instance of TaskExtractionAgent
func GetTaskExtractionAgent(config types.InternalAgentConfig) (*TaskExtractionAgent, error) {
	extractionOnce.Do(func() {
		extractionInstance = initializeTaskExtractionAgent(config)
	})
	return extractionInstance, nil
}

func initializeTaskExtractionAgent(config types.InternalAgentConfig) *TaskExtractionAgent {
	llmClient := NewLLMClient(&LLMConfig{
		Provider:      Qwen,
		BaseURL:       config.LLMConfig.BaseURL,
		APIKey:        config.LLMConfig.APIKey,
		Model:         config.LLMConfig.Model,
		Timeout:       30 * time.Second,
		Debug:         true,
		SystemMessage: "You are a specialized task extraction system that analyzes queries and produces structured task definitions.",
		Options: map[string]interface{}{
			"temperature": 0.3, // Lower temperature for more focused outputs
			"top_p":       0.9,
		},
	})

	promptMgr := NewPromptManager()
	if err := promptMgr.RegisterTemplate("taskExtractionPrompt", taskExtractionPromptTemplate); err != nil {
		panic(fmt.Sprintf("Failed to register task extraction prompt template: %v", err))
	}

	return &TaskExtractionAgent{
		llmClient: llmClient,
		promptMgr: promptMgr,
	}
}

// ExtractTask analyzes a user query and returns a structured task definition
func (a *TaskExtractionAgent) ExtractTask(ctx context.Context, query string) (string, error) {
	// Get available domains from the capability registry
	domains := capability.GetCapabilityRegistry().GetTopLevelCapabilities()

	// Prepare prompt data
	promptData := map[string]interface{}{
		"query":   query,
		"domains": domains,
	}

	// Generate the prompt
	prompt, err := a.promptMgr.GeneratePrompt("taskExtractionPrompt", promptData)
	if err != nil {
		return "", fmt.Errorf("generating prompt: %w", err)
	}

	// Get completion with retries
	completion, err := getCompletionWithRetries(ctx, a.llmClient, prompt)
	if err != nil {
		return "", fmt.Errorf("getting completion: %w", err)
	}

	log.Printf("task_extraction_agent response: %+v", completion)

	return completion, nil
}

// ExtractTaskWithRetry attempts to extract a task with retry logic
func (a *TaskExtractionAgent) ExtractTaskWithRetry(ctx context.Context, query string) (context.Context, error) {
	var lastErr error
	// Stopping for dev
	maxRetries := 1

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d for task extraction", attempt+1, maxRetries)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		// result, lastErr := a.ExtractTask(ctx, query)
		// Do not change
		result := `{
			"id": "task001",
			"title": "Build a REST API with Django Rest Framework",
			"description": "Develop a RESTful API using Python and the Django Rest Framework to handle HTTP requests and responses efficiently.",
			"requirements": {
				"skillPath": ["Web Development", "Back-end Development", "Python", "Django Rest Framework"],
				"action": "Build",
				"parameters": {
					"language": "Python",
					"framework": "Django Rest Framework",
					"apiType": "REST"
				}
			}
		}`

		if lastErr == nil && result != "" {
			// Create new context with the extraction result
			newCtx := context.WithValue(ctx, types.TaskExtractionResultKey, result)
			return newCtx, nil
		}

		log.Printf("Task extraction attempt %d failed: %v", attempt+1, lastErr)
	}

	return ctx, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
