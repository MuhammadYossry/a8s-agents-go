// metrics.go
package main

import (
	"strings"
	"sync"
	"time"
)

// MetricsData holds the counters and statistics for a specific task type
type MetricsData struct {
	TasksCompleted        int
	TasksFailed           int
	RoutingFailures       int
	RoutingSuccesses      int
	LastError             error
	LastErrorTime         time.Time
	TotalProcessingTime   time.Duration
	AverageProcessingTime time.Duration
	ProcessingTimes       []time.Duration
}

type MetricsKey struct {
	SkillPath string // Dot-separated path
	Action    string
}

type Metrics struct {
	mu             sync.RWMutex
	data           map[MetricsKey]*MetricsData
	taskStartTimes map[string]time.Time
}

func NewMetrics() *Metrics {
	return &Metrics{
		data:           make(map[MetricsKey]*MetricsData),
		taskStartTimes: make(map[string]time.Time),
	}
}

func createMetricsKey(req TaskRequirement) MetricsKey {
	return MetricsKey{
		SkillPath: strings.Join(req.SkillPath, "."),
		Action:    req.Action,
	}
}

// RecordTaskStart records the start time of a task
func (m *Metrics) RecordTaskStart(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskStartTimes[taskID] = time.Now()
}

// RecordTaskComplete updates metrics for a completed task
func (m *Metrics) RecordTaskComplete(requirements TaskRequirement, taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := createMetricsKey(requirements)
	if _, exists := m.data[key]; !exists {
		m.data[key] = &MetricsData{
			ProcessingTimes: make([]time.Duration, 0),
		}
	}

	if startTime, exists := m.taskStartTimes[taskID]; exists {
		duration := time.Since(startTime)
		m.data[key].ProcessingTimes = append(m.data[key].ProcessingTimes, duration)
		m.data[key].TotalProcessingTime += duration
		m.data[key].TasksCompleted++
		m.data[key].AverageProcessingTime = m.data[key].TotalProcessingTime /
			time.Duration(m.data[key].TasksCompleted)

		delete(m.taskStartTimes, taskID)
	} else {
		m.data[key].TasksCompleted++
	}
}

// initMetricsIfNeeded initializes metrics for a task type if not exists
func (m *Metrics) initMetricsIfNeeded(key MetricsKey) *MetricsData {
	if _, exists := m.data[key]; !exists {
		m.data[key] = &MetricsData{
			ProcessingTimes: make([]time.Duration, 0),
		}
	}
	return m.data[key]
}

// RecordTaskError updates metrics for a failed task
func (m *Metrics) RecordTaskError(requirements TaskRequirement, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := createMetricsKey(requirements)
	metrics := m.initMetricsIfNeeded(key)
	metrics.TasksFailed++
	metrics.LastError = err
	metrics.LastErrorTime = time.Now()
}

// RecordRoutingSuccess updates metrics for successful task routing
func (m *Metrics) RecordRoutingSuccess(requirements TaskRequirement, agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := createMetricsKey(requirements)
	metrics := m.initMetricsIfNeeded(key)
	metrics.RoutingSuccesses++
}

// GetMetrics returns the current metrics for a task type
func (m *Metrics) GetMetrics(requirements TaskRequirement) *MetricsData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := createMetricsKey(requirements)
	if metrics, exists := m.data[key]; exists {
		// Return a deep copy to prevent external modifications
		metricsCopy := *metrics
		if metrics.ProcessingTimes != nil {
			metricsCopy.ProcessingTimes = make([]time.Duration, len(metrics.ProcessingTimes))
			copy(metricsCopy.ProcessingTimes, metrics.ProcessingTimes)
		}
		return &metricsCopy
	}
	return nil
}

// RecordRoutingFailure updates metrics for failed task routing
func (m *Metrics) RecordRoutingFailure(requirements TaskRequirement, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := createMetricsKey(requirements)
	metrics := m.initMetricsIfNeeded(key)
	metrics.RoutingFailures++
}

// GetAllMetrics returns metrics for all task types
func (m *Metrics) GetAllMetrics() map[MetricsKey]*MetricsData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a deep copy to avoid external modifications
	result := make(map[MetricsKey]*MetricsData, len(m.data))
	for key, metrics := range m.data {
		metricsCopy := *metrics
		if metrics.ProcessingTimes != nil {
			metricsCopy.ProcessingTimes = make([]time.Duration, len(metrics.ProcessingTimes))
			copy(metricsCopy.ProcessingTimes, metrics.ProcessingTimes)
		}
		result[key] = &metricsCopy
	}
	return result
}

// ResetMetrics clears all metrics data
func (m *Metrics) ResetMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[MetricsKey]*MetricsData)
	m.taskStartTimes = make(map[string]time.Time)
}
