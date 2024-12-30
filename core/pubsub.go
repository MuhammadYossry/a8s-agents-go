// core/pubsub.go
package core

import (
	"context"
	"sync"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

type PubSub struct {
	mu     sync.RWMutex
	subs   map[string][]chan *types.Task
	closed bool
}

func NewPubSub() types.Broker {
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

	channels := ps.subs[topic]
	for _, ch := range channels {
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

	if ps.closed {
		return nil, nil
	}

	ch := make(chan *types.Task, 100)
	ps.subs[topic] = append(ps.subs[topic], ch)

	// Start a goroutine to clean up the channel when context is done
	go func() {
		<-ctx.Done()
		ps.Unsubscribe(context.Background(), topic, ch)
	}()

	return ch, nil
}

func (ps *PubSub) Unsubscribe(ctx context.Context, topic string, ch <-chan *types.Task) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	channels := ps.subs[topic]
	for i, existing := range channels {
		if existing == ch {
			// Remove the channel from the slice
			ps.subs[topic] = append(channels[:i], channels[i+1:]...)
			close(existing)
			break
		}
	}

	// Clean up empty topics
	if len(ps.subs[topic]) == 0 {
		delete(ps.subs, topic)
	}
	return nil
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
	ps.subs = make(map[string][]chan *types.Task)
	return nil
}
