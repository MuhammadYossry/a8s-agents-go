// internal/agents/agent_factory.go
package agents

import (
	"context"

	"github.com/MuhammadYossry/a8s-agents-go/types"
)

type AgentFactory struct{}

func NewAgentFactory() types.AgentFactory {
	return &AgentFactory{}
}

func (f *AgentFactory) GetPayloadAgent(ctx context.Context, config types.InternalAgentConfig) (types.PayloadAgent, error) {
	return GetPayloadAgent(ctx, config)
}

func (f *AgentFactory) GetTaskRoutingAgent(config types.InternalAgentConfig) (types.TaskRoutingAgent, error) {
	return GetTaskRoutingAgent(config)
}

func (f *AgentFactory) GetTaskExtractionAgent(config types.InternalAgentConfig) (types.TaskExtractionAgent, error) {
	return GetTaskExtractionAgent(config)
}
