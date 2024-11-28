package router

import (
    "time"

    "github.com/Relax-N-Tax/AgentNexus/internal/agent/types"
    "github.com/Relax-N-Tax/AgentNexus/internal/broker"
    "github.com/Relax-N-Tax/AgentNexus/internal/task/types"
)

// Task Distribution System
type TaskRouter struct {
    broker         *MessageBroker
    scheduler      *TaskScheduler
    loadBalancer   *LoadBalancer
}

func (tr *TaskRouter) RouteTask(task *Task) error {
    // Find capable agents
    agents := tr.findCapableAgents(task.Requirements)
    if len(agents) == 0 {
        return ErrNoCapableAgents
    }

    // Select best agent
    selectedAgent := tr.loadBalancer.SelectAgent(agents, task)
    if selectedAgent == nil {
        return ErrNoAvailableAgents
    }

    // Create task message
    msg := &Message{
        ID:        uuid.New().String(),
        Topic:     selectedAgent.AgentID,
        Payload:   task,
        Timestamp: time.Now(),
        Metadata:  map[string]string{
            "priority": strconv.Itoa(task.Priority),
            "deadline": task.Deadline.Format(time.RFC3339),
        },
    }

    // Publish task
    return tr.broker.PublishMessage(msg)
}