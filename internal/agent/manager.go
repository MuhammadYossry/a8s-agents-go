package agent

import (
    "sync"
    "fmt"

    "github.com/MuhammadYossry/AgentNexus/internal/agent/types"
    "github.com/MuhammadYossry/AgentNexus/pkg/monitoring"
)

// Agent Manager Implementation
type AgentManager struct {
    registry        *AgentRegistry
    heartbeat      *HeartbeatService
    capabilities   *CapabilityRegistry
    health         *HealthMonitor
}

func NewAgentManager(config ManagerConfig) *AgentManager {
    return &AgentManager{
        registry:      NewAgentRegistry(),
        heartbeat:    NewHeartbeatService(config.HeartbeatInterval),
        capabilities: NewCapabilityRegistry(),
        health:      NewHealthMonitor(config.HealthCheckInterval),
    }
}

// Agent Registration
func (am *AgentManager) RegisterAgent(agent *AIAgent) error {
    // Validate agent configuration
    if err := am.validateAgent(agent); err != nil {
        return fmt.Errorf("agent validation failed: %w", err)
    }

    // Register capabilities
    if err := am.capabilities.RegisterAgentCapabilities(agent); err != nil {
        return fmt.Errorf("capability registration failed: %w", err)
    }

    // Start health monitoring
    am.health.StartMonitoring(agent)

    // Initialize heartbeat
    am.heartbeat.RegisterAgent(agent)

    return am.registry.AddAgent(agent)
}