package broker

import (
    "context"
    "sync"
    "time"

    "go.uber.org/zap"
    "github.com/Relax-N-Tax/AgentNexus/internal/task"
)

type PubSubManager struct {
    mu            sync.RWMutex
    subscribers   map[string]map[string]chan *Message  // topic -> subscriberID -> channel
    logger        *zap.Logger
    closed        bool
    closeCh       chan struct{}
}

type Message struct {
    ID        string
    Topic     string
    Payload   interface{}
    Metadata  map[string]string
    Timestamp time.Time
}

type SubscribeOptions struct {
    BufferSize  int
    Timeout     time.Duration
}

func NewPubSubManager(logger *zap.Logger) *PubSubManager {
    return &PubSubManager{
        subscribers: make(map[string]map[string]chan *Message),
        logger:     logger,
        closeCh:    make(chan struct{}),
    }
}

func (ps *PubSubManager) Subscribe(topic, subscriberID string, opts SubscribeOptions) (<-chan *Message, error) {
    ps.mu.Lock()
    defer ps.mu.Unlock()

    if ps.closed {
        return nil, ErrManagerClosed
    }

    if opts.BufferSize <= 0 {
        opts.BufferSize = 100 // default buffer size
    }

    if _, exists := ps.subscribers[topic]; !exists {
        ps.subscribers[topic] = make(map[string]chan *Message)
    }

    if _, exists := ps.subscribers[topic][subscriberID]; exists {
        return nil, ErrDuplicateSubscriber
    }

    ch := make(chan *Message, opts.BufferSize)
    ps.subscribers[topic][subscriberID] = ch

    ps.logger.Info("new subscriber registered",
        zap.String("topic", topic),
        zap.String("subscriber_id", subscriberID),
    )

    return ch, nil
}

func (ps *PubSubManager) Unsubscribe(topic, subscriberID string) error {
    ps.mu.Lock()
    defer ps.mu.Unlock()

    if _, exists := ps.subscribers[topic]; !exists {
        return ErrTopicNotFound
    }

    if ch, exists := ps.subscribers[topic][subscriberID]; exists {
        close(ch)
        delete(ps.subscribers[topic], subscriberID)
        if len(ps.subscribers[topic]) == 0 {
            delete(ps.subscribers, topic)
        }
        return nil
    }

    return ErrSubscriberNotFound
}

func (ps *PubSubManager) Publish(ctx context.Context, msg *Message) error {
    ps.mu.RLock()
    defer ps.mu.RUnlock()

    if ps.closed {
        return ErrManagerClosed
    }

    subscribers, exists := ps.subscribers[msg.Topic]
    if !exists {
        return nil // No subscribers for this topic
    }

    for subID, ch := range subscribers {
        select {
        case ch <- msg:
            // Message sent successfully
        default:
            ps.logger.Warn("subscriber channel full, dropping message",
                zap.String("topic", msg.Topic),
                zap.String("subscriber_id", subID),
            )
        }
    }

    return nil
}

func (ps *PubSubManager) PublishWithTimeout(ctx context.Context, msg *Message, timeout time.Duration) error {
    timer := time.NewTimer(timeout)
    defer timer.Stop()

    done := make(chan error, 1)
    go func() {
        done <- ps.Publish(ctx, msg)
    }()

    select {
    case err := <-done:
        return err
    case <-timer.C:
        return ErrPublishTimeout
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (ps *PubSubManager) Close() error {
    ps.mu.Lock()
    defer ps.mu.Unlock()

    if ps.closed {
        return ErrManagerClosed
    }

    ps.closed = true
    close(ps.closeCh)

    // Close all subscriber channels
    for topic, subscribers := range ps.subscribers {
        for subID, ch := range subscribers {
            close(ch)
            delete(subscribers, subID)
        }
        delete(ps.subscribers, topic)
    }

    return nil
}

// Error definitions
var (
    ErrManagerClosed       = errors.New("pubsub manager is closed")
    ErrTopicNotFound      = errors.New("topic not found")
    ErrSubscriberNotFound = errors.New("subscriber not found")
    ErrDuplicateSubscriber = errors.New("subscriber already exists for this topic")
    ErrPublishTimeout     = errors.New("publish timeout")
)