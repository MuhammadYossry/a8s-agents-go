// prompt_manager.go
package agents

import (
	"bytes"
	"fmt"
	"text/template"
)

type PromptManager struct {
	templates map[string]*template.Template
}

func NewPromptManager() *PromptManager {
	return &PromptManager{
		templates: make(map[string]*template.Template),
	}
}

func (pm *PromptManager) RegisterTemplate(name, templateStr string) error {
	tmpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	pm.templates[name] = tmpl
	return nil
}

func (pm *PromptManager) GeneratePrompt(templateName string, data interface{}) (string, error) {
	tmpl, exists := pm.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

// XXX: Predefined templates
// We need in the future to have a prompt store and auto prompt generation/caching
const generateCodePromptTemplate = `System Message:
You are an API request generator for a Python code generation system. Generate valid JSON payloads that match the schema exactly. Return only the JSON payload without explanations.

Task Information:
- Type: Code Generation
- Description: {{.task.Description}}
- Skills Required: {{.task.Requirements.SkillPath}}
- Framework: {{index .task.Requirements.Parameters "framework"}}
- Additional Info: {{.task.Title}}

Required Request Schema:
{{.action.InputSchema}}

Example Valid Request:
{{index .action.Examples.ValidRequests 0}}

Agent Capabilities Context:
The agent has expertise in:
{{range .capabilities}}
- {{.SkillPath}}: {{.Metadata}}
{{end}}

Generate a valid JSON payload following the schema and example structure, incorporating the task requirements and agent capabilities. The response should be only the JSON payload, with no additional text or explanations.`

const deployPreviewPromptTemplate = `System Message:
You are an API request generator for a Python deployment system. Generate valid JSON payloads that match the schema exactly. Return only the JSON payload without explanations.

Task Information:
- Type: Deploy Preview
- Description: {{.task.Description}}
- Branch: {{index .task.Requirements.Parameters "branch"}}
- Environment: {{index .task.Requirements.Parameters "environment"}}

Required Request Schema:
{{.action.InputSchema}}

Example Valid Request:
{{index .action.Examples.ValidRequests 0}}

Generate a valid JSON payload following the schema and example structure, adapting it to the task details provided.`
