package core

import (
	"context"
	"sync"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

type Broker interface {
	Publish(ctx context.Context, topic string, task *types.Task) error
	Subscribe(ctx context.Context, topic string) (<-chan *types.Task, error)
	Close() error
}

type PubSub struct {
	mu     sync.RWMutex
	subs   map[string][]chan *types.Task
	closed bool
}

func NewPubSub() *PubSub {
	return &PubSub{
		subs: make(map[string][]chan *types.Task),
	}
}

func (ps *PubSub) Publish(ctx context.Context, topic string, task *types.Task) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.closed {
		return nil
	}

	for _, ch := range ps.subs[topic] {
		select {
		case ch <- task:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Skip if channel is full
		}
	}
	return nil
}

func (ps *PubSub) Subscribe(ctx context.Context, topic string) (<-chan *types.Task, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ch := make(chan *types.Task, 100)
	ps.subs[topic] = append(ps.subs[topic], ch)
	return ch, nil
}

func (ps *PubSub) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return nil
	}

	ps.closed = true
	for _, channels := range ps.subs {
		for _, ch := range channels {
			close(ch)
		}
	}
	return nil
}
