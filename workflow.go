// workflow.go
package main

import (
	"context"
	"fmt"
	"log"
)

type WorkflowExecutor struct {
	broker   Broker
	registry *CapabilityRegistry
	metrics  *Metrics
	agents   []*Agent
}

func NewWorkflowExecutor(broker Broker, registry *CapabilityRegistry, metrics *Metrics, agents []*Agent) *WorkflowExecutor {
	return &WorkflowExecutor{
		broker:   broker,
		registry: registry,
		metrics:  metrics,
		agents:   agents,
	}
}

func (we *WorkflowExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	// Record start time for metrics
	we.metrics.RecordTaskStart(task.ID)

	log.Printf("Workflow starting execution of task: %s", task.Title)

	// Execute sub-tasks using workflow's agents
	// TODO: Orchestrate task between agents
	for _, subtask := range we.createSubtasks(task) {
		// Find matching agent for subtask
		matchingAgents := we.findMatchingAgents(subtask)
		if len(matchingAgents) == 0 {
			return nil, fmt.Errorf("no matching agents for subtask: %s", subtask.Title)
		}

		// Assign to first matching agent
		selectedAgent := matchingAgents[0]
		if err := we.broker.Publish(ctx, string(selectedAgent), subtask); err != nil {
			return nil, fmt.Errorf("failed to assign subtask to agent: %w", err)
		}
	}

	// For now, simulate successful completion
	result := &TaskResult{
		TaskID:  task.ID,
		Success: true,
	}

	// Record completion in metrics
	we.metrics.RecordTaskComplete(task.Type, task.ID)

	return result, nil
}

func (we *WorkflowExecutor) findMatchingAgents(task *Task) []AgentID {
	var matchingAgents []AgentID
	for _, agent := range we.agents {
		// Check if agent supports task type
		supportsType := false
		for _, t := range agent.taskTypes {
			if t == task.Type {
				supportsType = true
				break
			}
		}
		if !supportsType {
			continue
		}

		// Check if agent has required skills
		skills := agent.skillsByType[task.Type]
		hasSkills := true
		for _, required := range task.SkillsRequired {
			found := false
			for _, available := range skills {
				if required == available {
					found = true
					break
				}
			}
			if !found {
				hasSkills = false
				break
			}
		}

		if hasSkills {
			matchingAgents = append(matchingAgents, agent.id)
		}
	}
	return matchingAgents
}

func (we *WorkflowExecutor) createSubtasks(task *Task) []*Task {
	// This is a simplified version - in a real system, you'd likely have more sophisticated
	// task decomposition logic based on the workflow type and task requirements
	return []*Task{
		{
			ID:             fmt.Sprintf("%s-subtask1", task.ID),
			Title:          fmt.Sprintf("%s - Processing", task.Title),
			Type:           task.Type,
			SkillsRequired: task.SkillsRequired,
			Status:         TaskStatusPending,
		},
	}
}

type Workflow struct {
	id           WorkFlowID
	broker       Broker
	executor     *WorkflowExecutor
	metrics      *Metrics
	registry     *CapabilityRegistry
	agents       []*Agent
	taskTypes    []string
	skillsByType map[string][]string
	cancelFunc   context.CancelFunc
}

func NewWorkflow(
	id WorkFlowID,
	broker Broker,
	metrics *Metrics,
	registry *CapabilityRegistry,
	agents []*Agent,
	taskTypes []string,
	skillsByType map[string][]string,
) *Workflow {
	executor := NewWorkflowExecutor(broker, registry, metrics, agents)
	return &Workflow{
		id:           id,
		broker:       broker,
		executor:     executor,
		metrics:      metrics,
		registry:     registry,
		agents:       agents,
		taskTypes:    taskTypes,
		skillsByType: skillsByType,
	}
}

func (w *Workflow) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	w.cancelFunc = cancel

	// Register workflow's capabilities
	w.registry.RegisterWorkflow(w.id, WorkFlowCapability{
		WorkFlowID:   w.id,
		TaskTypes:    w.taskTypes,
		SkillsByType: w.skillsByType,
		Resources:    map[string]int{"cpu": 4, "memory": 8},
	})

	// Subscribe to workflow-specific topic
	taskCh, err := w.broker.Subscribe(ctx, string(w.id))
	if err != nil {
		return err
	}

	// Start task processing
	go func() {
		for {
			select {
			case task := <-taskCh:
				if task == nil {
					return
				}
				go w.processTask(ctx, task)
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Printf("Workflow %s started with capabilities - Task Types: %v", w.id, w.taskTypes)
	return nil
}

func (w *Workflow) processTask(ctx context.Context, task *Task) {
	log.Printf("Workflow %s processing task: %s", w.id, task.Title)

	result, err := w.executor.Execute(ctx, task)
	if err != nil {
		log.Printf("Workflow %s failed to execute task %s: %v", w.id, task.Title, err)
		w.metrics.RecordTaskError(task.Type, err)
		return
	}

	if result.Success {
		log.Printf("Workflow %s successfully completed task: %s", w.id, task.Title)
	} else {
		log.Printf("Workflow %s task completed but unsuccessful: %s", w.id, task.Title)
	}
}

func (w *Workflow) Stop(ctx context.Context) error {
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
	return nil
}
