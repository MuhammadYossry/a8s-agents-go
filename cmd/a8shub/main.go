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

// AgentDef with flexible structure for UI components and workflows
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
	Workflows    json.RawMessage        `json:"workflows,omitempty"`
}

type Capability struct {
	SkillPath []string               `json:"skill_path"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Action with flexible UI components
type Action struct {
	Name                string          `json:"name"`
	Slug                string          `json:"slug"`
	ActionType          string          `json:"actionType"`
	Path                string          `json:"path"`
	Method              string          `json:"method"`
	InputSchema         json.RawMessage `json:"inputSchema"`
	OutputSchema        json.RawMessage `json:"outputSchema"`
	Examples            json.RawMessage `json:"examples,omitempty"`
	Description         string          `json:"description"`
	IsMDResponseEnabled bool            `json:"isMDResponseEnabled"`
	ResponseTemplateMD  string          `json:"responseTemplateMD,omitempty"`
	UIComponents        json.RawMessage `json:"uiComponents,omitempty"`
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

func handlePush(cfg *config, filePath string) error {
	// Read raw file data instead of unmarshaling to our struct
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Just do basic validation without full unmarshal
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("invalid JSON data: %w", err)
	}

	// Validate only essential fields
	slug, ok := jsonData["slug"].(string)
	if !ok || slug == "" {
		return fmt.Errorf("agent definition must contain a slug")
	}

	version, ok := jsonData["version"].(string)
	if !ok || version == "" {
		return fmt.Errorf("agent definition must contain a version")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("agentfile", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	// Write raw data to form instead of marshaling our struct
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("failed to write data to form: %w", err)
	}
	writer.Close()

	resp, err := sendRequest(cfg, "POST", "/agents/push", writer.FormDataContentType(), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("Successfully pushed %s:%s\n", slug, version)
	return nil
}

func handlePull(cfg *config, ref string) error {
	slug, version, err := parseReference(ref)
	if err != nil {
		return err
	}

	resp, err := sendRequest(cfg, "GET", fmt.Sprintf("/agents/pull/%s/%s", slug, version), "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read raw response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse just enough to get the agent definition
	var result map[string]json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	agentDefData, ok := result["agent_def"]
	if !ok {
		return fmt.Errorf("response does not contain agent_def field")
	}

	outputFile := fmt.Sprintf("%s-%s.json", slug, version)

	// Pretty-print the JSON before saving
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, agentDefData, "", "  "); err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	if err := os.WriteFile(outputFile, prettyJSON.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Successfully pulled %s:%s to %s\n", slug, version, outputFile)
	return nil
}

func handleList(cfg *config) error {
	resp, err := sendRequest(cfg, "GET", "/agents/list", "", nil)
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
