package definationloader

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

// MarkdownFormatter handles formatting and parsing of markdown documentation
type MarkdownFormatter struct {
	buffer         strings.Builder
	parsedSections *types.DocSection
	errorResponses string // Store error responses separately
}

// NewMarkdownFormatter creates a new markdown formatter instance
func NewMarkdownFormatter() *MarkdownFormatter {
	return &MarkdownFormatter{
		parsedSections: &types.DocSection{
			Title:    "root",
			Sections: make(map[string]*types.DocSection),
		},
	}
}

// MarkDownFromConfig generates markdown documentation from agent configuration
func (g *MarkdownFormatter) MarkDownFromConfig(config *types.AgentConfig) (string, error) {
	g.writeHeader()

	for _, agent := range config.Agents {
		g.writeAgent(&agent)
	}

	// Add error responses at the end
	g.writeErrorResponses()

	return g.buffer.String(), nil
}

// ParseSections parses the markdown into structured sections
func (g *MarkdownFormatter) ParseSections(markdown string) *types.DocSection {
	lines := strings.Split(markdown, "\n")
	root := &types.DocSection{
		Title:    "root",
		Sections: make(map[string]*types.DocSection),
	}

	var currentSection *types.DocSection
	var currentSubSection *types.DocSection
	var content strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// New main section
			title := strings.TrimPrefix(line, "## ")
			currentSection = &types.DocSection{
				Title:    title,
				Sections: make(map[string]*types.DocSection),
			}
			root.Sections[title] = currentSection
			currentSubSection = nil
		} else if strings.HasPrefix(line, "### ") {
			// New subsection
			if currentSection != nil {
				title := strings.TrimPrefix(line, "### ")
				currentSubSection = &types.DocSection{
					Title:    title,
					Sections: make(map[string]*types.DocSection),
				}
				currentSection.Sections[title] = currentSubSection
				content.Reset()
			}
		} else if currentSubSection != nil {
			content.WriteString(line + "\n")
			currentSubSection.Content = content.String()
		} else if currentSection != nil {
			content.WriteString(line + "\n")
			currentSection.Content = content.String()
		}
	}

	return root
}

// GetSection retrieves a specific section by path
func (g *MarkdownFormatter) GetSection(path ...string) (*types.DocSection, error) {
	if err := g.validateSectionPath(path...); err != nil {
		return nil, err
	}

	current := g.parsedSections
	for _, segment := range path {
		if section, ok := current.Sections[segment]; ok {
			current = section
		} else {
			return nil, fmt.Errorf("section not found: %s", strings.Join(path, " → "))
		}
	}
	return current, nil
}

// ListSections returns a slice of all available section paths
func (g *MarkdownFormatter) ListSections() [][]string {
	sections := make([][]string, 0)
	g.collectSections(g.parsedSections, []string{}, &sections)
	return sections
}

// collectSections recursively collects all section paths
func (g *MarkdownFormatter) collectSections(current *types.DocSection, path []string, sections *[][]string) {
	if current == nil {
		return
	}

	if len(path) > 0 {
		// Create a copy of the current path
		currentPath := make([]string, len(path))
		copy(currentPath, path)
		*sections = append(*sections, currentPath)
	}

	for title, section := range current.Sections {
		newPath := append(path, title)
		g.collectSections(section, newPath, sections)
	}
}

// validateSectionPath checks if a section path exists
func (g *MarkdownFormatter) validateSectionPath(path ...string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty section path")
	}

	availableSections := g.ListSections()
	pathStr := strings.Join(path, " → ")

	// Convert path to comparable format
	targetPath := make([]string, len(path))
	copy(targetPath, path)

	for _, section := range availableSections {
		if len(section) == len(path) {
			match := true
			for i := range section {
				if section[i] != path[i] {
					match = false
					break
				}
			}
			if match {
				return nil
			}
		}
	}

	return fmt.Errorf("invalid section path: %s. Available sections: %v",
		pathStr, formatAvailableSections(availableSections))
}

// formatAvailableSections formats section paths for error messages
func formatAvailableSections(sections [][]string) string {
	paths := make([]string, len(sections))
	for i, section := range sections {
		paths[i] = strings.Join(section, " → ")
	}
	return strings.Join(paths, ", ")
}

// Helper functions for writing markdown content

func (g *MarkdownFormatter) writeHeader() {
	g.writeLine("# AI Agents Service Documentation\n")
	g.writeLine("Welcome to our AI Agents service documentation. This service hosts several AI agents, each providing specific capabilities through well-documented endpoints.\n")
}

func (g *MarkdownFormatter) writeAgent(agent *types.AgentDefinition) {
	g.writeLine(fmt.Sprintf("## %s\n", agent.ID))
	g.writeLine(fmt.Sprintf("**Description:** %s\n", agent.Description))
	g.writeLine(fmt.Sprintf("**Base URL:** `%s`\n", agent.BaseURL))

	if len(agent.Capabilities) > 0 {
		g.writeCapabilities(agent.Capabilities)
	}

	if len(agent.Actions) > 0 {
		g.writeActions(agent.Actions)
	}
}

func (g *MarkdownFormatter) writeCapabilities(capabilities []types.Capability) {
	g.writeLine("## Capabilities\n")
	g.writeLine("The following sections detail the specific capabilities of this agent:\n")

	for _, cap := range capabilities {
		g.writeCapability(cap)
	}
}

func (g *MarkdownFormatter) writeCapability(cap types.Capability) {
	g.writeLine(fmt.Sprintf("### %s\n", strings.Join(cap.SkillPath, " → ")))
	g.writeLine("**Capability Details:**")

	for key, value := range cap.Metadata {
		switch v := value.(type) {
		case string:
			g.writeLine(fmt.Sprintf("- %s: %s", key, v))
		case []interface{}:
			items := make([]string, len(v))
			for i, item := range v {
				items[i] = fmt.Sprintf("%v", item)
			}
			g.writeLine(fmt.Sprintf("- %s: %s", key, strings.Join(items, ", ")))
		}
	}
	g.writeLine("")
}

func (g *MarkdownFormatter) writeActions(actions []types.Action) {
	g.writeLine("## Available Endpoints\n")
	g.writeLine("This section describes all available endpoints for interacting with the agent:\n")

	for _, action := range actions {
		g.writeAction(action)
	}
}

func (g *MarkdownFormatter) writeAction(action types.Action) {
	g.writeLine(fmt.Sprintf("### %s\n", action.Name))
	g.writeLine(fmt.Sprintf("**Endpoint:** `%s %s`\n", action.Method, action.Path))

	g.writeSchema("Input Schema", action.InputSchema)
	g.writeSchema("Output Schema", action.OutputSchema)
}

func (g *MarkdownFormatter) writeSchema(title string, schema types.SchemaConfig) {
	g.writeLine(fmt.Sprintf("#### %s\n", title))

	if schema.Description != "" {
		g.writeLine(fmt.Sprintf("%s\n", schema.Description))
	}

	if len(schema.Properties) > 0 {
		g.writeLine("**Properties:**\n")
		g.writeProperties(schema.Properties, schema.Required)
	}

	g.writeLine("")
}

func (g *MarkdownFormatter) writeProperties(props map[string]types.Property, required []string) {
	for name, prop := range props {
		g.writeProperty(name, prop, contains(required, name))
	}
}

func (g *MarkdownFormatter) writeProperty(name string, prop types.Property, required bool) {
	g.writeLine(fmt.Sprintf("- `%s`: ", name))
	g.writeLine(fmt.Sprintf("  * Type: `%s`", g.formatType(prop)))

	if required {
		g.writeLine("  * Required: Yes")
	}

	if prop.Default != nil {
		g.writeLine(fmt.Sprintf("  * Default: `%v`", prop.Default))
	}

	g.writeConstraints(prop)
	g.writeLine("")
}

func (g *MarkdownFormatter) formatType(prop types.Property) string {
	if len(prop.Enum) > 0 {
		return fmt.Sprintf("%s (enum: %s)", prop.Type, strings.Join(prop.Enum, ", "))
	}
	if prop.Const != "" {
		return fmt.Sprintf("%s (const: %s)", prop.Type, prop.Const)
	}
	return prop.Type
}

func (g *MarkdownFormatter) writeConstraints(prop types.Property) {
	constraints := []string{}

	if prop.Minimum != nil {
		constraints = append(constraints, fmt.Sprintf("minimum: %v", *prop.Minimum))
	}
	if prop.Maximum != nil {
		constraints = append(constraints, fmt.Sprintf("maximum: %v", *prop.Maximum))
	}

	if len(constraints) > 0 {
		g.writeLine("  * Constraints: " + strings.Join(constraints, ", "))
	}
}

func (g *MarkdownFormatter) writeErrorResponses() {
	if g.errorResponses == "" {
		g.writeLine("\n### Error Responses\n")

		errorCodes := map[string]string{
			"400": "Bad Request - Invalid input parameters",
			"401": "Unauthorized - Authentication required",
			"403": "Forbidden - Insufficient permissions",
			"404": "Not Found - Resource not found",
			"422": "Unprocessable Entity - Validation error",
			"500": "Internal Server Error - Server-side error occurred",
		}

		for code, description := range errorCodes {
			g.writeLine(fmt.Sprintf("**Status %s**: %s", code, description))
			example := map[string]interface{}{
				"error": map[string]interface{}{
					"code":    code,
					"message": description,
					"details": "Additional error context would appear here",
				},
			}
			jsonBytes, _ := json.MarshalIndent(example, "", "  ")
			g.writeLine(fmt.Sprintf("Example:\n```json\n%s\n```\n", string(jsonBytes)))
		}

		g.errorResponses = g.buffer.String()
	}
}

func (g *MarkdownFormatter) writeLine(text string) {
	g.buffer.WriteString(text + "\n")
}

// Helper function
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
