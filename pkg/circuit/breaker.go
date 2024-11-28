package circuit

import (
	"sync"
	"time"
)

// Circuit Breaker Implementation
type CircuitBreaker struct {
    mu           sync.RWMutex
    failures     int
    lastFailure  time.Time
    state        string // closed, open, half-open
    threshold    int
    resetTimeout time.Duration
}

func NewCircuitBreaker() *CircuitBreaker {
    return &CircuitBreaker{
        threshold:    5,
        resetTimeout: 30 * time.Second,
        state:       "closed",
    }
}

func (cb *CircuitBreaker) AllowRequest() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()

    if cb.state == "closed" {
        return true
    }

    if cb.state == "open" {
        if time.Since(cb.lastFailure) > cb.resetTimeout {
            cb.state = "half-open"
            return true
        }
        return false
    }

    return cb.state == "half-open"
}
