package broker

import (
    "github.com/Relax-N-Tax/AgentNexus/internal/agent/types"
    "github.com/Relax-N-Tax/AgentNexus/internal/task/types"
)

// Message Broker System
type MessageBroker struct {
    topics    map[string]*Topic
    queues    map[string]*TaskQueue
    pubsub    *PubSubManager
}

func NewMessageBroker(logger *zap.Logger) *MessageBroker {
    return &MessageBroker{
        topics: make(map[string]*Topic),
        queues: make(map[string]*TaskQueue),
        pubsub: NewPubSubManager(logger),
    }
}