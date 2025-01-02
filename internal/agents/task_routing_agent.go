// task_routing_prompt.go
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	// *Error*

	"github.com/Relax-N-Tax/AgentNexus/definationloader"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

// TaskRoutingAgent determines the most suitable agent and action for a given task
type TaskRoutingAgent struct {
	llmClient *LLMClient
	promptMgr *PromptManager
}

const taskRoutingPromptTemplate = `You are an advanced Task Orchestration System responsible for intelligent task routing and agent coordination. Your primary function is to analyze incoming tasks and match them with the most suitable agent and action based on their capabilities.

SYSTEM CONTEXT:
{{- if .AgentsData }}
{{ .AgentsData }}
{{- else }}
Available Agents and Their Capabilities:
{
    "python-code-agent": {
        "type": "service",
        "capabilities": [
            {
                "skillPath": ["Development"],
                "level": "domain",
                "specialization": 0.7
            },
            {
                "skillPath": ["Development", "Backend", "Python"],
                "level": "specialty",
                "specialization": 0.9,
                "metadata": {
                    "frameworks": ["Django", "FastAPI"],
                    "versions": {
                        "python": ">=3.7"
                    }
                }
            }
        ],
        "actions": {
            "generateCode": {
                "parameters": ["specification", "framework"],
                "availability": 1.0
            },
            "improveCode": {
                "parameters": ["code", "requirements"],
                "availability": 1.0
            },
            "testCode": {
                "parameters": ["code", "testRequirements"],
                "availability": 1.0
            }
        }
    }
}
{{- end }}

MATCHING ALGORITHM:

1. Capability Matching (40% of total score)
- Direct skill path matching
- Framework and technology alignment
- Version compatibility
- Specialization level consideration

2. Action Suitability (30% of total score)
- Action availability
- Parameter compatibility
- Historical performance
- Required vs available features

3. Agent Performance (20% of total score)
- Historical success rate
- Average response time
- Error rate
- Specialization score

4. Context Alignment (10% of total score)
- Domain-specific requirements
- Special constraints
- Environmental factors

TASK TO ANALYZE:
{{- if .TaskJsonData }}
{{ .TaskJsonData }}
{{- else }}
{
    "id": "fallback_task",
    "title": "{{ .Task.Title }}",
    "description": "{{ .Task.Description }}",
    "requirements": {
        "skillPath": {{ .Task.Requirements.SkillPath }},
        "action": "{{ .Task.Requirements.Action }}",
        "parameters": {{ .Task.Requirements.Parameters }}
    }
}
{{- end }}

RESPONSE SCHEMA:
{
    "matched": boolean,
    "match": {
        "agentId": string,          // ID of the selected agent
        "action": string,           // Selected action from agent's available actions
        "confidence": number,       // 0-100 score
        "matchDetails": {
            "pathMatchScore": number,   // 0-40
            "frameworkScore": number,   // 0-20
            "actionScore": number,      // 0-20
            "versionScore": number      // 0-20
        },
        "reasoning": string
    } | null,
    "alternatives": [
        {
            "agentId": string,
            "confidence": number,
            "reason": string
        }
    ],
    "error": string | null
}

EVALUATION CRITERIA:

1. Required Match Conditions:
- Complete skill path match OR hierarchical parent match
- Action availability and parameter compatibility
- Minimum confidence score of 60

2. Scoring Components:
- Skill Path Match (0-40 points)
   * Exact path match: 40
   * Parent path match: 30
   * Partial path match: 20
   * Domain-only match: 10

- Framework/Tech Support (0-20 points)
   * Full support: 20
   * Partial support: 10
   * Basic compatibility: 5

- Action Compatibility (0-20 points)
   * Full parameter match: 20
   * Partial parameter match: 10
   * Basic action match: 5

- Version Compatibility (0-20 points)
   * Exact version match: 20
   * Compatible version: 15
   * Upgradable version: 10

3. Rejection Criteria:
- No skill path match
- Missing required action
- Confidence score < 60
- Critical parameter mismatch

Please analyze the provided task and determine the optimal agent and action match based on these criteria. Return your analysis in the specified JSON format.`

// Types for the prompt template
type TaskRoutingPromptData struct {
	AgentsData   string      // Agents capabilities and definitions
	TaskJsonData string      // Task data in JSON format
	Task         *types.Task // Fallback task data if TaskJsonData is empty
}

var (
	taskRoutingAgent *TaskRoutingAgent
	routingAgentOnce sync.Once
)

func GetTaskRoutingAgent(config types.InternalAgentConfig) (*TaskRoutingAgent, error) {
	routingAgentOnce.Do(func() {
		taskRoutingAgent = initializeTaskRoutingAgent(config)
	})
	return taskRoutingAgent, nil
}

func initializeTaskRoutingAgent(config types.InternalAgentConfig) *TaskRoutingAgent {
	llmClient, _ := NewLLMClient(&LLMConfig{
		Provider:      Qwen,
		BaseURL:       config.LLMConfig.BaseURL,
		APIKey:        config.LLMConfig.APIKey,
		Model:         config.LLMConfig.Model,
		Timeout:       90 * time.Second,
		Debug:         true,
		SystemMessage: "You are an advanced Task Orchestration System responsible for intelligent task routing and agent coordination.", // Added system message
		Options: map[string]interface{}{
			"temperature":   0.3, // Lower temperature for more deterministic matching
			"top_p":         0.9,
			"result_format": "message",
			"stream":        false,
		},
	})

	promptMgr := NewPromptManager()
	if err := promptMgr.RegisterTemplate("taskRoutingPrompt", taskRoutingPromptTemplate); err != nil {
		panic(fmt.Sprintf("Failed to register task routing prompt template: %v", err))
	}

	return &TaskRoutingAgent{
		llmClient: llmClient,
		promptMgr: promptMgr,
	}
}

func PrepareRoutingPromptData(ctx context.Context, task *types.Task) TaskRoutingPromptData {
	var agentData string

	// Try to get the markdown content from context first
	if mdContent, ok := ctx.Value(types.AgentsMarkDownKey).(string); ok && mdContent != "" {
		log.Println("task-routing-agent: Found markdown content")

		// Create formatter from the markdown content
		mdFormatter := definationloader.NewMarkdownFormatter()
		parsedRoot := mdFormatter.ParseSections(mdContent)

		// Print full section tree for debugging
		printMarkdownSections("task-routing-agent: ", parsedRoot, 0)

		var formattedContent strings.Builder

		// Try to get each main section
		for title, section := range parsedRoot.Sections {
			log.Printf("task-routing-agent: Processing section: %s", title)
			switch title {
			case "Agent", "Capabilities", "Available Endpoints":
				log.Printf("task-routing-agent: Found %s section", title)
				formattedContent.WriteString(fmt.Sprintf("### %s\n", title))
				formattedContent.WriteString(section.Content + "\n\n")
			}
		}

		if formattedContent.Len() > 0 {
			agentData = formattedContent.String()
		} else {
			// If no sections were found, use the complete markdown
			agentData = mdContent
		}
	}

	// Fallback to raw data if no markdown content available
	if agentData == "" {
		log.Println("task-routing-agent: Falling back to raw agents data")
		if rawData := ctx.Value(types.RawAgentsDataKey); rawData != nil {
			if str, ok := rawData.(string); ok {
				agentData = str
			}
		}
	}

	log.Printf("task-routing-agent: Final agent data length: %d", len(agentData))
	if len(agentData) > 200 {
		log.Printf("task-routing-agent: Content preview: %s...", agentData[:200])
	}

	return TaskRoutingPromptData{
		AgentsData:   agentData,
		TaskJsonData: getTaskDataFromContext(ctx, task),
		Task:         task,
	}
}

// Helper function to print all markdown sections recursively
func printMarkdownSections(prefix string, section *types.DocSection, depth int) {
	if section == nil {
		return
	}

	indent := strings.Repeat("  ", depth)
	log.Printf("%s%s%s: %d bytes content", prefix, indent, section.Title, len(section.Content))

	if len(section.Content) > 0 {
		preview := section.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		log.Printf("%s%s  Content Preview: %s", prefix, indent, preview)
	}

	for title, subsection := range section.Sections {
		log.Printf("%s%s  └─ Subsection: %s", prefix, indent, title)
		printMarkdownSections(prefix, subsection, depth+1)
	}
}

// Helper function to get task data from context
func getTaskDataFromContext(ctx context.Context, _ *types.Task) string {
	// Try to get from context first
	if taskData := ctx.Value(types.TaskExtractionResultKey); taskData != nil {
		if str, ok := taskData.(string); ok && str != "" {
			return str
		}
	}

	// Return empty string to use template fallback
	return ""
}

func (a *TaskRoutingAgent) FindMatchingAgent(ctx context.Context, task *types.Task) (*types.ActionPlan, error) {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Prepare the prompt data
	promptData := PrepareRoutingPromptData(ctx, task)

	// Generate prompt
	prompt, err := a.promptMgr.GeneratePrompt("taskRoutingPrompt", promptData)
	if err != nil {
		return nil, fmt.Errorf("generating prompt: %w", err)
	}

	log.Printf("Generated routing prompt for task %s", task.ID)

	// Get LLM completion with retries
	var completion string
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d for task routing", attempt+1, maxRetries)
			time.Sleep(time.Duration(attempt) * 2 * time.Second) // Exponential backoff
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while getting completion: %w", ctx.Err())
		default:
			// Use CreateChatCompletion instead of GetCompletion
			completion, err = a.llmClient.CreateChatCompletion(ctx, prompt)
			if err == nil {
				break
			}
			lastErr = err
			log.Printf("Routing attempt %d failed: %v", attempt+1, err)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("getting completion after %d attempts: %w", maxRetries, lastErr)
	}

	// Validate completion
	if completion == "" {
		return nil, fmt.Errorf("received empty completion from LLM")
	}

	// Parse the match result
	var matchResult types.MatchResult
	if err := json.Unmarshal([]byte(completion), &matchResult); err != nil {
		// Log the completion for debugging
		log.Printf("Failed to parse completion: %s", completion)
		return nil, fmt.Errorf("parsing match result: %w", err)
	}

	// Validate match result
	if err := validateMatchResult(&matchResult); err != nil {
		return nil, fmt.Errorf("invalid match result: %w", err)
	}

	// Convert to ActionPlan
	actionPlan := convertMatchResultToActionPlan(&matchResult)

	// Log the result
	if matchResult.Matched && matchResult.Match != nil {
		log.Printf("Found matching agent for task %s: agent=%s, action=%s, confidence=%.2f",
			task.ID, matchResult.Match.AgentID, matchResult.Match.Action, matchResult.Match.Confidence)
	} else {
		log.Printf("No matching agent found for task %s: %s", task.ID, matchResult.Error)
	}

	return actionPlan, nil
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
