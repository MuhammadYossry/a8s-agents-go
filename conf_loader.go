package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/internal/agents"
	"github.com/Relax-N-Tax/AgentNexus/types"
	"gopkg.in/yaml.v2"
)

type Config struct {
	LLM    agents.LLMConfig       `yaml:"llm"`
	Agents []types.AgentDefConfig `yaml:"agents"`
}

func NewConfig() (*Config, error) {
	llmConfig, err := loadLLMConfig("a8s_llm.conf")
	if err != nil {
		return nil, err
	}

	agents, err := loadAgentDefinitions("agents.a8s")
	if err != nil {
		log.Printf("WARNING: Failed to load agents: %v", err)
	}

	return &Config{
		LLM:    llmConfig,
		Agents: agents,
	}, nil
}

func loadConfigFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var filteredLines []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			filteredLines = append(filteredLines, line)
		}
	}

	return []byte(strings.Join(filteredLines, "\n")), nil
}

func loadLLMConfig(filename string) (agents.LLMConfig, error) {
	config := agents.LLMConfig{
		Provider: agents.Qwen,
		Model:    "Qwen-2.5-72B-Chat",
		Timeout:  30 * time.Second,
	}

	data, err := loadConfigFile(filename)
	if err == nil {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return config, fmt.Errorf("failed to parse LLM config: %w", err)
		}
	}

	// Environment fallbacks
	if config.BaseURL == "" {
		config.BaseURL = os.Getenv("RNT_OPENAI_URL")
	}
	if config.APIKey == "" {
		config.APIKey = os.Getenv("RNT_OPENAI_API_KEY")
	}

	return config, nil
}

func loadAgentDefinitions(filename string) ([]types.AgentDefConfig, error) {
	var agents []types.AgentDefConfig
	data, err := os.ReadFile(filename)
	if err != nil {
		return agents, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		agents = append(agents, types.AgentDefConfig{
			Name:    strings.TrimSpace(parts[0]),
			Version: strings.TrimSpace(parts[1]),
		})
	}

	return agents, nil
}
