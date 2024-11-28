package agent

import (
    "time"

    "github.com/Relax-N-Tax/AgentNexus/internal/task/types"
)

// Core Agent Types
type AIAgent struct {
    AgentID        string         `json:"agentId"`
    Name           string         `json:"name"`
    Version        string         `json:"version"`
    Capabilities   []Capability   `json:"capabilities"`
    Status         AgentStatus    `json:"status"`
    Config         AgentConfig    `json:"config"`
    Metrics        *AgentMetrics  `json:"metrics"`
}

type Capability struct {
    Name           string         `json:"name"`
    Version        string         `json:"version"`
    Properties     PropertySet    `json:"properties"`
    Requirements   Requirements   `json:"requirements"`
}

type AgentStatus struct {
    State          string         `json:"state"`      // online, offline, busy, degraded
    HealthScore    float64        `json:"healthScore"`
    LastHeartbeat  time.Time      `json:"lastHeartbeat"`
    CurrentLoad    float64        `json:"currentLoad"`
}