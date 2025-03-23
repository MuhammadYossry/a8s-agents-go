package definationloader

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MuhammadYossry/a8s-agents-go/types"
)

const (
	ActionTypeTalk     = "talk"
	ActionTypeGenerate = "generate"
)

// MarkdownFormatter handles formatting and parsing of markdown documentation
type MarkdownFormatter struct {
	buffer         strings.Builder
	parsedSections *types.DocSection
	errorResponses string
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

// MarkDownFromJSON generates markdown format of a single agent definition from JSON
func (g *MarkdownFormatter) MarkDownFromJSON(jsonData []byte) (string, error) {
	var agent types.AgentDefinition
	if err := json.Unmarshal(jsonData, &agent); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	return g.MarkDownFromAgent(agent)
}

// MarkDownFromConfig generates markdown documentation from agent configuration
func (g *MarkdownFormatter) MarkDownFromConfig(config interface{}) (string, error) {
	agentBytes, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %v", err)
	}

	var agent types.AgentDefinition
	if err := json.Unmarshal(agentBytes, &agent); err != nil {
		return "", fmt.Errorf("failed to unmarshal agent: %v", err)
	}

	return g.MarkDownFromAgent(agent)
}

// MarkDownFromAgent generates markdown format of a single agent definition
func (g *MarkdownFormatter) MarkDownFromAgent(agent types.AgentDefinition) (string, error) {
	g.buffer = strings.Builder{} // Reset buffer

	// Write agent header
	g.writeLine(fmt.Sprintf("# %s\n", agent.ID))
	g.writeLine(fmt.Sprintf("%s\n", agent.Description))
	g.writeLine(fmt.Sprintf("**Base URL:** `%s`\n", agent.BaseURL))

	// Write capabilities
	if len(agent.Capabilities) > 0 {
		g.writeCapabilities(agent.Capabilities)
	}

	// Write actions grouped by type
	if len(agent.Actions) > 0 {
		g.writeActionsByType(agent.Actions)
	}

	// Add error responses
	g.writeErrorResponses()

	return g.buffer.String(), nil
}

func (g *MarkdownFormatter) writeCapabilities(capabilities []types.Capability) {
	g.writeLine("## Capabilities\n")

	// Group capabilities by domain
	domainCaps := make(map[string][]types.Capability)
	for _, cap := range capabilities {
		if len(cap.SkillPath) > 0 {
			domain := cap.SkillPath[0]
			domainCaps[domain] = append(domainCaps[domain], cap)
		}
	}

	// Write each domain's capabilities
	for domain, caps := range domainCaps {
		g.writeLine(fmt.Sprintf("### %s\n", domain))
		for _, cap := range caps {
			g.writeCapability(cap)
		}
	}
}

func (g *MarkdownFormatter) writeCapability(cap types.Capability) {
	if len(cap.SkillPath) > 1 {
		skillPath := strings.Join(cap.SkillPath[1:], " → ")
		g.writeLine(fmt.Sprintf("#### %s\n", skillPath))
	}

	g.writeLine(fmt.Sprintf("- Level: `%s`", cap.Level))
	g.writeMetadata(cap.Metadata)
	g.writeLine("")
}

func (g *MarkdownFormatter) writeMetadata(metadata map[string]interface{}) {
	for key, value := range metadata {
		switch v := value.(type) {
		case string:
			g.writeLine(fmt.Sprintf("- %s: `%s`", key, v))
		case []interface{}:
			items := make([]string, len(v))
			for i, item := range v {
				items[i] = fmt.Sprintf("%v", item)
			}
			g.writeLine(fmt.Sprintf("- %s: `%s`", key, strings.Join(items, "`, `")))
		}
	}
}

func (g *MarkdownFormatter) writeActionsByType(actions []types.Action) {
	g.writeLine("## Actions\n")

	// Group actions by type
	talkActions := []types.Action{}
	generateActions := []types.Action{}
	otherActions := []types.Action{}

	for _, action := range actions {
		switch action.ActionType {
		case ActionTypeTalk:
			talkActions = append(talkActions, action)
		case ActionTypeGenerate:
			generateActions = append(generateActions, action)
		default:
			otherActions = append(otherActions, action)
		}
	}

	// Write conversation actions
	if len(talkActions) > 0 {
		g.writeLine("### Conversation Actions\n")
		g.writeLine("These actions enable interaction through natural language.\n")
		for _, action := range talkActions {
			g.writeAction(action)
		}
	}

	// Write generation actions
	if len(generateActions) > 0 {
		g.writeLine("### Generation Actions\n")
		g.writeLine("These actions generate or transform content.\n")
		for _, action := range generateActions {
			g.writeAction(action)
		}
	}

	// Write other actions
	if len(otherActions) > 0 {
		g.writeLine("### Other Actions\n")
		for _, action := range otherActions {
			g.writeAction(action)
		}
	}
}

func (g *MarkdownFormatter) writeAction(action types.Action) {
	g.writeLine(fmt.Sprintf("#### %s\n", action.Name))
	g.writeLine(fmt.Sprintf("**Endpoint:** `%s %s`\n", action.Method, action.Path))

	// Input Schema
	if len(action.InputSchema.Properties) > 0 {
		g.writeLine("##### Input Schema\n")
		g.writeSchemaDetails(action.InputSchema)
	}

	// Output Schema
	if len(action.OutputSchema.Properties) > 0 {
		g.writeLine("##### Output Schema\n")
		g.writeSchemaDetails(action.OutputSchema)
	}

	g.writeLine("")
}

func (g *MarkdownFormatter) writeSchemaDetails(schema types.SchemaConfig) {
	if schema.Description != "" {
		g.writeLine(fmt.Sprintf("%s\n", schema.Description))
	}

	g.writeLine("**Properties:**\n")
	for name, prop := range schema.Properties {
		g.writePropertyDetails(name, prop, schema.Required)
	}
	g.writeLine("")
}

func (g *MarkdownFormatter) writePropertyDetails(name string, prop types.Property, required []string) {
	isRequired := contains(required, name)
	details := []string{fmt.Sprintf("type: `%s`", prop.Type)}

	if prop.Description != "" {
		details = append(details, fmt.Sprintf("description: %s", prop.Description))
	}
	if prop.Format != "" {
		details = append(details, fmt.Sprintf("format: `%s`", prop.Format))
	}
	if len(prop.Enum) > 0 {
		details = append(details, fmt.Sprintf("enum: `%s`", strings.Join(prop.Enum, "`, `")))
	}
	if prop.Pattern != "" {
		details = append(details, fmt.Sprintf("pattern: `%s`", prop.Pattern))
	}
	if prop.Default != nil {
		details = append(details, fmt.Sprintf("default: `%v`", prop.Default))
	}

	requiredMark := ""
	if isRequired {
		requiredMark = " (required)"
	}

	g.writeLine(fmt.Sprintf("- `%s`%s: %s", name, requiredMark, strings.Join(details, ", ")))

	// Handle nested properties
	if len(prop.Properties) > 0 {
		g.writeLine("  Nested properties:")
		for subName, subProp := range prop.Properties {
			g.writePropertyDetails(subName, subProp, prop.Required)
		}
	}

	// Handle array items
	if prop.Items != nil {
		g.writeLine("  Array items:")
		g.writePropertyDetails("item", *prop.Items, nil)
	}
}

func (g *MarkdownFormatter) writeErrorResponses() {
	g.writeLine("\n## Error Responses\n")
	g.writeLine("All endpoints follow standard HTTP status codes and include detailed error messages.\n")

	errorCodes := map[string]string{
		"400": "Bad Request - Invalid input parameters",
		"401": "Unauthorized - Authentication required",
		"403": "Forbidden - Insufficient permissions",
		"404": "Not Found - Resource not found",
		"422": "Unprocessable Entity - Validation error",
		"500": "Internal Server Error - Server-side error occurred",
	}

	for code, description := range errorCodes {
		g.writeLine(fmt.Sprintf("**%s**: %s\n", code, description))
	}
}

// ParseSections parses the markdown into structured sections
func (g *MarkdownFormatter) ParseSections(markdown string) *types.DocSection {
	lines := strings.Split(markdown, "\n")
	root := &types.DocSection{
		Title:    "root",
		Sections: make(map[string]*types.DocSection),
	}

	var currentSection *types.DocSection
	var content strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// New section
			title := strings.TrimPrefix(line, "## ")
			currentSection = &types.DocSection{
				Title:    title,
				Sections: make(map[string]*types.DocSection),
			}
			root.Sections[title] = currentSection
			content.Reset()
		} else if currentSection != nil {
			content.WriteString(line + "\n")
			currentSection.Content = content.String()
		}
	}

	return root
}

// GetSection retrieves a specific section by path
func (g *MarkdownFormatter) GetSection(path ...string) (*types.DocSection, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty section path")
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

func (g *MarkdownFormatter) collectSections(current *types.DocSection, path []string, sections *[][]string) {
	if current == nil {
		return
	}

	if len(path) > 0 {
		currentPath := make([]string, len(path))
		copy(currentPath, path)
		*sections = append(*sections, currentPath)
	}

	for title, section := range current.Sections {
		newPath := append(path, title)
		g.collectSections(section, newPath, sections)
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
