package hub

import (
	"time"
)

// Config holds configuration for the AgentsHub
type Config struct {
	Address string        // Server address (e.g., ":8080")
	BaseURL string        // Base URL for the server
	Timeout time.Duration // Operation timeout
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		Address: ":8082",
		BaseURL: "http://localhost:8082",
		Timeout: 30 * time.Second,
	}
}

// AgentFile represents the metadata and content of an agent
type AgentFile struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata"`
	CreateTime int64             `json:"create_time"`
}

// DefinationRegistry defines the interface for agent defination registry
type DefinationRegistry interface {
	Store(name, version string, agent *AgentFile) error
	Get(name, version string) (*AgentFile, error)
	GetJSON(name, version string) ([]byte, error)
	Close() error
}
