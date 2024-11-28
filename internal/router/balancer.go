package router

import (
    "sync"
    "time"
    "math/rand"

    "github.com/MuhammadYossry/AgentNexus/internal/agent/types"
    "github.com/MuhammadYossry/AgentNexus/internal/task/types"
)

type LoadBalancer struct {
    mu             sync.RWMutex
    strategy       string                // round-robin, least-loaded, random
    lastAgentIndex int                  // for round-robin
    stats          map[string]AgentStats // agent performance stats
}

type AgentStats struct {
    CurrentLoad    float64
    SuccessRate    float64
    ResponseTime   time.Duration
    LastUpdated    time.Time
}

func NewLoadBalancer(strategy string) *LoadBalancer {
    return &LoadBalancer{
        strategy: strategy,
        stats:    make(map[string]AgentStats),
    }
}

func (lb *LoadBalancer) SelectAgent(agents []*agent.AIAgent, task *task.TaskInstance) *agent.AIAgent {
    lb.mu.Lock()
    defer lb.mu.Unlock()

    if len(agents) == 0 {
        return nil
    }

    switch lb.strategy {
    case "least-loaded":
        return lb.selectLeastLoaded(agents)
    case "round-robin":
        return lb.selectRoundRobin(agents)
    default:
        return lb.selectRandom(agents)
    }
}

func (lb *LoadBalancer) selectLeastLoaded(agents []*agent.AIAgent) *agent.AIAgent {
    var selected *agent.AIAgent
    minLoad := float64(1.0)

    for _, agent := range agents {
        if stats, exists := lb.stats[agent.AgentID]; exists {
            if stats.CurrentLoad < minLoad {
                minLoad = stats.CurrentLoad
                selected = agent
            }
        }
    }

    return selected
}

func (lb *LoadBalancer) selectRoundRobin(agents []*agent.AIAgent) *agent.AIAgent {
    lb.lastAgentIndex = (lb.lastAgentIndex + 1) % len(agents)
    return agents[lb.lastAgentIndex]
}

func (lb *LoadBalancer) selectRandom(agents []*agent.AIAgent) *agent.AIAgent {
    return agents[rand.Intn(len(agents))]
}

func (lb *LoadBalancer) UpdateStats(agentID string, stats AgentStats) {
    lb.mu.Lock()
    defer lb.mu.Unlock()
    
    stats.LastUpdated = time.Now()
    lb.stats[agentID] = stats
}