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
