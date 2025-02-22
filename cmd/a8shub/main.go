package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/spf13/cobra"
)

// Agent represents an agent
type Agent struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Slug      string    `json:"slug"`
    Versions  []string  `json:"versions"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// AgentDefinition represents a versioned agent definition
type AgentDefinition struct {
    ID         int       `json:"id"`
    AgentID    int       `json:"agent_id"`
    Version    string    `json:"version"`
    Definition AgentDef  `json:"definition"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}

// AgentDef represents the actual agent definition content
type AgentDef struct {
    Name         string                 `json:"name"`
    Slug         string                 `json:"slug"`
    Version      string                 `json:"version"`
    Type         string                 `json:"type"`
    Description  string                 `json:"description"`
    BaseURL      string                 `json:"baseUrl"`
    MetaInfo     map[string]interface{} `json:"metaInfo,omitempty"`
    Capabilities []Capability           `json:"capabilities"`
    Actions      []Action               `json:"actions"`
    Workflows    []Workflow             `json:"workflows,omitempty"`
}

type Capability struct {
    SkillPath []string               `json:"skill_path"`
    Metadata  map[string]interface{} `json:"metadata"`
}

type Workflow struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    Description string         `json:"description"`
    Steps       []WorkflowStep `json:"steps"`
    InitialStep string         `json:"initial_step"`
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
    ID          string           `json:"id"`
    Type        string           `json:"type"` // start, action, end
    Action      *string          `json:"action"`
    Transitions []StepTransition `json:"transitions"`
}

// StepTransition defines how to move from one step to another
type StepTransition struct {
    Target    string  `json:"target"`
    Condition *string `json:"condition"`
}

type Action struct {
    Name                string                 `json:"name"`
    Slug                string                 `json:"slug"`
    ActionType          string                 `json:"actionType"`
    Path                string                 `json:"path"`
    Method              string                 `json:"method"`
    InputSchema         map[string]interface{} `json:"inputSchema"`
    OutputSchema        map[string]interface{} `json:"outputSchema"`
    Examples            map[string]interface{} `json:"examples,omitempty"`
    Description         string                 `json:"description"`
    IsMDResponseEnabled bool                   `json:"isMDResponseEnabled"`
    ResponseTemplateMD  string                 `json:"responseTemplateMD,omitempty"`
    UIComponents       []UIComponent          `json:"uiComponents,omitempty"`
}

// UIComponent represents a UI component definition
type UIComponent struct {
    Type    string                 `json:"type"`
    Key     string                 `json:"key"`
    Title   string                 `json:"title"`
    Meta    map[string]interface{} `json:"meta,omitempty"`
    State   map[string]interface{} `json:"state,omitempty"`
    Fields  []UIField             `json:"fields,omitempty"`
    Columns []UIColumn            `json:"columns,omitempty"`
    Data    []interface{}         `json:"data,omitempty"`
    Actions []string              `json:"actions,omitempty"`
    Pagination bool               `json:"pagination,omitempty"`
    PageSize   int                `json:"page_size,omitempty"`
}

// UIField represents a form field in a UI component
type UIField struct {
    Name     string     `json:"name"`
    Label    string     `json:"label"`
    Type     string     `json:"type"`
    Required bool       `json:"required"`
    Options  []UIOption `json:"options,omitempty"`
}

// UIOption represents an option in a select/radio field
type UIOption struct {
    Value string `json:"value"`
    Label string `json:"label"`
}

// UIColumn represents a column in a table component
type UIColumn struct {
    Field    string `json:"field"`
    Header   string `json:"header"`
    Sortable bool   `json:"sortable"`
}

type config struct {
    serverURL string
}

func newConfig() *config {
    return &config{
        serverURL: "http://localhost:8082/api/v1",
    }
}

func main() {
    if err := newRootCmd().Execute(); err != nil {
        log.Fatal(err)
    }
}

func newRootCmd() *cobra.Command {
    cfg := newConfig()

    rootCmd := &cobra.Command{
        Use:   "a8shub",
        Short: "AgentsHub CLI tool",
        Long:  `CLI for managing AgentsHub agents. Supports push, pull, and list operations.`,
    }

    rootCmd.PersistentFlags().StringVar(&cfg.serverURL, "server", cfg.serverURL, "AgentsHub server URL")
    rootCmd.AddCommand(
        newPushCmd(cfg),
        newPullCmd(cfg),
        newListCmd(cfg),
        newShowCmd(cfg),
    )

    return rootCmd
}

func newPushCmd(cfg *config) *cobra.Command {
    return &cobra.Command{
        Use:   "push [file]",
        Short: "Push an agent file using metadata from the agent definition",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return handlePush(cfg, args[0])
        },
    }
}

func newPullCmd(cfg *config) *cobra.Command {
    return &cobra.Command{
        Use:   "pull [slug:version]",
        Short: "Pull an agent",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return handlePull(cfg, args[0])
        },
    }
}

func newListCmd(cfg *config) *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all agents",
        RunE: func(cmd *cobra.Command, args []string) error {
            return handleList(cfg)
        },
    }
}

func newShowCmd(cfg *config) *cobra.Command {
    return &cobra.Command{
        Use:   "show [slug:version]",
        Short: "Show agent details",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return handleShow(cfg, args[0])
        },
    }
}

func handlePush(cfg *config, filePath string) error {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return fmt.Errorf("failed to read file: %w", err)
    }

    var agentDef AgentDef
    if err := json.Unmarshal(data, &agentDef); err != nil {
        return fmt.Errorf("invalid agent definition: %w", err)
    }

    if agentDef.Slug == "" {
        return fmt.Errorf("agent definition must contain a slug")
    }
    if agentDef.Version == "" {
        return fmt.Errorf("agent definition must contain a version")
    }

    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    part, err := writer.CreateFormFile("agentfile", filepath.Base(filePath))
    if err != nil {
        return fmt.Errorf("failed to create form file: %w", err)
    }

    if err := json.NewEncoder(part).Encode(agentDef); err != nil {
        return fmt.Errorf("failed to encode agent definition: %w", err)
    }
    writer.Close()

    resp, err := sendRequest(cfg, "POST", "/agent/push", writer.FormDataContentType(), body)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    fmt.Printf("Successfully pushed %s:%s\n", agentDef.Slug, agentDef.Version)
    return nil
}

func handlePull(cfg *config, ref string) error {
    slug, version, err := parseReference(ref)
    if err != nil {
        return err
    }

    resp, err := sendRequest(cfg, "GET", fmt.Sprintf("/agent/pull/%s/%s", slug, version), "", nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var result struct {
        AgentDef AgentDef `json:"agent_def"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("failed to decode response: %w", err)
    }

    outputFile := fmt.Sprintf("%s-%s.json", slug, version)
    data, err := json.MarshalIndent(result.AgentDef, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal agent: %w", err)
    }

    if err := os.WriteFile(outputFile, data, 0644); err != nil {
        return fmt.Errorf("failed to write file: %w", err)
    }

    fmt.Printf("Successfully pulled %s:%s to %s\n", slug, version, outputFile)
    return nil
}

func handleList(cfg *config) error {
    resp, err := sendRequest(cfg, "GET", "/agent/list", "", nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var agents []struct {
        Name        string   `json:"name"`
        Slug        string   `json:"slug"`
        Versions    []string `json:"versions"`
        Description string   `json:"description"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
        return fmt.Errorf("failed to decode response: %w", err)
    }

    for _, agent := range agents {
        latestVersion := agent.Versions[len(agent.Versions)-1]
        fmt.Printf("%s:%s - %s\nVersions: %s\n\n",
            agent.Slug,
            latestVersion,
            agent.Description,
            strings.Join(agent.Versions, ", "))
    }
    return nil
}

func handleShow(cfg *config, ref string) error {
    slug, _, err := parseReference(ref)
    if err != nil {
        return err
    }

    resp, err := sendRequest(cfg, "GET", fmt.Sprintf("/agent/show/%s", slug), "", nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var result struct {
        AgentDef AgentDef `json:"agent_def"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("failed to decode response: %w", err)
    }

    fmt.Print(generateMarkdown(&result.AgentDef))
    return nil
}

func parseReference(ref string) (string, string, error) {
    parts := strings.Split(ref, ":")
    if len(parts) != 2 {
        return "", "", fmt.Errorf("invalid reference format. Use slug:version")
    }
    return parts[0], parts[1], nil
}

func sendRequest(cfg *config, method, path string, contentType string, body io.Reader) (*http.Response, error) {
    url := fmt.Sprintf("%s%s", cfg.serverURL, path)
    req, err := http.NewRequest(method, url, body)
    if err != nil {
        return nil, err
    }

    if contentType != "" {
        req.Header.Set("Content-Type", contentType)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode >= 400 {
        body, _ := io.ReadAll(resp.Body)
        var errResp struct {
            Detail string `json:"detail"`
        }
        if err := json.Unmarshal(body, &errResp); err != nil {
            return nil, fmt.Errorf("%s", string(body))
        }
        return nil, fmt.Errorf("%s", errResp.Detail)
    }

    return resp, nil
}

func generateUIComponentMarkdown(component *UIComponent) string {
    var b strings.Builder

    fmt.Fprintf(&b, "#### %s Component (%s)\n", component.Title, component.Type)
    fmt.Fprintf(&b, "- **Key:** %s\n", component.Key)
    
    if len(component.Meta) > 0 {
        fmt.Fprintf(&b, "- **Metadata:**\n")
        if meta, err := json.MarshalIndent(component.Meta, "  ", "  "); err == nil {
            fmt.Fprintf(&b, "  ```json\n  %s\n  ```\n", string(meta))
        }
    }

    if len(component.Fields) > 0 {
        fmt.Fprintf(&b, "- **Fields:**\n")
        for _, field := range component.Fields {
            fmt.Fprintf(&b, "  - %s (%s)\n", field.Label, field.Type)
            if field.Required {
                fmt.Fprintf(&b, "    - Required: true\n")
            }
            if len(field.Options) > 0 {
                fmt.Fprintf(&b, "    - Options:\n")
                for _, opt := range field.Options {
                    fmt.Fprintf(&b, "      - %s: %s\n", opt.Value, opt.Label)
                }
            }
        }
    }

    if len(component.Columns) > 0 {
        fmt.Fprintf(&b, "- **Columns:**\n")
        for _, col := range component.Columns {
            sortable := ""
            if col.Sortable {
                sortable = " (sortable)"
            }
            fmt.Fprintf(&b, "  - %s: %s%s\n", col.Field, col.Header, sortable)
        }
    }

    if len(component.Actions) > 0 {
        fmt.Fprintf(&b, "- **Actions:** %s\n", strings.Join(component.Actions, ", "))
    }

    if component.Pagination {
        fmt.Fprintf(&b, "- **Pagination:** Enabled (Page Size: %d)\n", component.PageSize)
    }

    return b.String()
}
func generateMarkdown(def *AgentDef) string {
    var b strings.Builder

    fmt.Fprintf(&b, "# %s (v%s)\n\n", def.Name, def.Version)
    fmt.Fprintf(&b, "**Slug:** %s\n", def.Slug)
    fmt.Fprintf(&b, "**Type:** %s\n", def.Type)
    fmt.Fprintf(&b, "**Description:** %s\n", def.Description)
    fmt.Fprintf(&b, "**Base URL:** %s\n\n", def.BaseURL)

    if len(def.Capabilities) > 0 {
        fmt.Fprintf(&b, "## Capabilities\n\n")
        for _, cap := range def.Capabilities {
            fmt.Fprintf(&b, "### %s\n", strings.Join(cap.SkillPath, " > "))
            if meta, err := json.MarshalIndent(cap.Metadata, "", "  "); err == nil {
                fmt.Fprintf(&b, "```json\n%s\n```\n\n", string(meta))
            }
        }
    }

    if len(def.Actions) > 0 {
        fmt.Fprintf(&b, "## Actions\n\n")
        for _, action := range def.Actions {
            fmt.Fprintf(&b, "### %s\n", action.Name)
            fmt.Fprintf(&b, "**Slug:** %s\n", action.Slug)
            fmt.Fprintf(&b, "**Type:** %s\n", action.ActionType)
            fmt.Fprintf(&b, "**Path:** %s\n", action.Path)
            fmt.Fprintf(&b, "**Method:** %s\n", action.Method)
            fmt.Fprintf(&b, "**Description:** %s\n\n", action.Description)

            fmt.Fprintf(&b, "#### Input Schema\n")
            if input, err := json.MarshalIndent(action.InputSchema, "", "  "); err == nil {
                fmt.Fprintf(&b, "```json\n%s\n```\n\n", string(input))
            }

            fmt.Fprintf(&b, "#### Output Schema\n")
            if output, err := json.MarshalIndent(action.OutputSchema, "", "  "); err == nil {
                fmt.Fprintf(&b, "```json\n%s\n```\n\n", string(output))
            }

            if action.Examples != nil {
                fmt.Fprintf(&b, "#### Examples\n")
                if examples, err := json.MarshalIndent(action.Examples, "", "  "); err == nil {
                    fmt.Fprintf(&b, "```json\n%s\n```\n\n", string(examples))
                }
            }
            if len(action.UIComponents) > 0 {
                fmt.Fprintf(&b, "#### UI Components\n\n")
                for _, component := range action.UIComponents {
                    fmt.Fprintf(&b, "%s\n", generateUIComponentMarkdown(&component))
                }
                fmt.Fprintf(&b, "\n")
            }
        }
    }

    return b.String()
}
