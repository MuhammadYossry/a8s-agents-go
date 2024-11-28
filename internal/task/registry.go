package engine

import (
    "sync"
    "fmt"

    "github.com/Relax-N-Tax/AgentNexus/internal/task/types"
)

// Registry Interface
type TaskRegistry interface {
    RegisterDefinition(def *AITaskDefinition) error
    GetDefinition(id string) (*AITaskDefinition, error)
    ListDefinitions(tags []string) ([]*AITaskDefinition, error)
    UpdateDefinition(def *AITaskDefinition) error
}

// In-Memory Registry Implementation
type InMemoryTaskRegistry struct {
    mu           sync.RWMutex
    definitions  map[string]*AITaskDefinition
}

func NewInMemoryTaskRegistry() *InMemoryTaskRegistry {
    return &InMemoryTaskRegistry{
        definitions: make(map[string]*AITaskDefinition),
    }
}

func (r *InMemoryTaskRegistry) RegisterDefinition(def *AITaskDefinition) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.definitions[def.TaskID]; exists {
        return fmt.Errorf("task definition %s already exists", def.TaskID)
    }

    r.definitions[def.TaskID] = def
    return nil
}

func (r *InMemoryTaskRegistry) GetDefinition(id string) (*AITaskDefinition, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    def, exists := r.definitions[id]
    if !exists {
        return nil, fmt.Errorf("task definition %s not found", id)
    }
    return def, nil
}

func (r *InMemoryTaskRegistry) ListDefinitions(tags []string) ([]*AITaskDefinition, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    if len(tags) == 0 {
        defs := make([]*AITaskDefinition, 0, len(r.definitions))
        for _, def := range r.definitions {
            defs = append(defs, def)
        }
        return defs, nil
    }

    tagSet := make(map[string]bool)
    for _, tag := range tags {
        tagSet[tag] = true
    }

    var defs []*AITaskDefinition
    for _, def := range r.definitions {
        for _, tag := range def.Tags {
            if tagSet[tag] {
                defs = append(defs, def)
                break
            }
        }
    }
    return defs, nil
}

func (r *InMemoryTaskRegistry) UpdateDefinition(def *AITaskDefinition) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.definitions[def.TaskID]; !exists {
        return fmt.Errorf("task definition %s not found", def.TaskID)
    }

    r.definitions[def.TaskID] = def
    return nil
}