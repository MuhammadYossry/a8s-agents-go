package agents

import (
	"sync"
	"time"
)

type Memory struct {
	shortTerm []MemoryItem
	longTerm  []MemoryItem
	maxSize   int
	mu        sync.RWMutex
}

type MemoryItem struct {
	Type    string
	Content interface{}
	Time    int64
}

func NewMemory(maxSize int) *Memory {
	return &Memory{
		shortTerm: make([]MemoryItem, 0),
		longTerm:  make([]MemoryItem, 0),
		maxSize:   maxSize,
		mu:        sync.RWMutex{},
	}
}

// AddMemory adds a new memory item
func (m *Memory) AddMemory(itemType string, content interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := MemoryItem{
		Type:    itemType,
		Content: content,
		Time:    time.Now().Unix(),
	}

	// Add to short-term memory
	m.shortTerm = append(m.shortTerm, item)

	// If short-term memory is full, move oldest items to long-term memory
	if len(m.shortTerm) > m.maxSize {
		// Move the oldest item to long-term memory
		m.longTerm = append(m.longTerm, m.shortTerm[0])
		m.shortTerm = m.shortTerm[1:]
	}
}

// GetRecentMemories retrieves recent memories of a specific type
func (m *Memory) GetRecentMemories(itemType string, limit int) []MemoryItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var memories []MemoryItem
	for i := len(m.shortTerm) - 1; i >= 0 && len(memories) < limit; i-- {
		if m.shortTerm[i].Type == itemType {
			memories = append(memories, m.shortTerm[i])
		}
	}

	return memories
}

// SearchMemories searches both short-term and long-term memories
func (m *Memory) SearchMemories(itemType string) []MemoryItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []MemoryItem

	// Search short-term memory
	for _, item := range m.shortTerm {
		if item.Type == itemType {
			results = append(results, item)
		}
	}

	// Search long-term memory
	for _, item := range m.longTerm {
		if item.Type == itemType {
			results = append(results, item)
		}
	}

	return results
}
