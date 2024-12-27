// task_extraction_agent.go
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/capability"
	"github.com/Relax-N-Tax/AgentNexus/types"
	"github.com/google/uuid"
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
- Task paths follow a hierarchical structure: Domain -> Subdomain -> Technology -> Action.
- Common domains include: {{.domains}}

TASK REQUIREMENTS:
1. First, identify if the input is a task request.
2. Then, extract the core components: action, technology, framework, and any specific requirements.
3. Finally, generate a structured JSON response.

THOUGHT PROCESS (internal only):
1. Is this a task request?
2. What is the primary action being requested?
3. What technology stack is involved?
4. What domain and subdomain does this belong to?
5. Are there any specific parameters or requirements?

FORMAT SPECIFICATION:
{
    "id": "taskXXX",  // Generated UUID
    "title": "",      // Concise task title
    "description": "", // Detailed task description
    "requirements": {
        "skillPath": [], // Hierarchical path array
        "action": "",    // Primary action
        "parameters": {} // Additional parameters
    }
}

CONSTRAINTS:
- Output must be valid JSON only
- No explanatory text before or after JSON
- All fields must be present
- Skill paths must follow the hierarchical structure
- IDs should be unique

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
		SkillPath  []string               `json:"skillPath"`
		Action     string                 `json:"action"`
		Parameters map[string]interface{} `json:"parameters"`
	} `json:"requirements"`
}

// GetTaskExtractionAgent returns a singleton instance of TaskExtractionAgent
func GetTaskExtractionAgent(ctx context.Context, config types.InternalAgentConfig) (*TaskExtractionAgent, error) {
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
func (a *TaskExtractionAgent) ExtractTask(ctx context.Context, query string) (*ExtractionResult, error) {
	// Generate a new UUID for the task
	taskID := fmt.Sprintf("task-%s", uuid.New().String()[:8])

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
		return nil, fmt.Errorf("generating prompt: %w", err)
	}

	// Get completion with retries
	completion, err := getCompletionWithRetries(ctx, a.llmClient, prompt)
	if err != nil {
		return nil, fmt.Errorf("getting completion: %w", err)
	}

	log.Printf("task_extraction_agent response: %+v", completion)

	// Parse and validate the response
	var result ExtractionResult
	if err := json.Unmarshal([]byte(completion), &result); err != nil {
		return nil, fmt.Errorf("parsing extraction result: %w", err)
	}

	// Validate the result
	if err := validateExtractionResult(&result); err != nil {
		return nil, fmt.Errorf("validating extraction result: %w", err)
	}

	// Ensure we use the generated task ID
	result.ID = taskID

	return &result, nil
}

// ExtractTaskWithRetry attempts to extract a task with retry logic
func (a *TaskExtractionAgent) ExtractTaskWithRetry(ctx context.Context, query string) (*ExtractionResult, error) {
	var result *ExtractionResult
	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d for task extraction", attempt+1, maxRetries)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		result, lastErr = a.ExtractTask(ctx, query)
		if lastErr == nil && result != nil {
			return result, nil
		}

		log.Printf("Task extraction attempt %d failed: %v", attempt+1, lastErr)
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func validateExtractionResult(result *ExtractionResult) error {
	if result.Title == "" {
		return fmt.Errorf("missing title")
	}
	if result.Description == "" {
		return fmt.Errorf("missing description")
	}
	if len(result.Requirements.SkillPath) == 0 {
		return fmt.Errorf("missing skill path")
	}
	if result.Requirements.Action == "" {
		return fmt.Errorf("missing action")
	}
	if result.Requirements.Parameters == nil {
		return fmt.Errorf("missing parameters")
	}
	return nil
}
