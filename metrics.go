// metrics.go
package main

import (
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

type Metrics struct {
	mu             sync.RWMutex
	data           map[string]*MetricsData
	taskStartTimes map[string]time.Time
}

func NewMetrics() *Metrics {
	return &Metrics{
		data:           make(map[string]*MetricsData),
		taskStartTimes: make(map[string]time.Time),
	}
}

// RecordTaskStart records the start time of a task
func (m *Metrics) RecordTaskStart(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskStartTimes[taskID] = time.Now()
}

// RecordTaskComplete updates metrics for a completed task
func (m *Metrics) RecordTaskComplete(taskType string, taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.data[taskType]; !exists {
		m.data[taskType] = &MetricsData{
			ProcessingTimes: make([]time.Duration, 0),
		}
	}

	// Calculate duration if we have a start time
	if startTime, exists := m.taskStartTimes[taskID]; exists {
		duration := time.Since(startTime)
		m.data[taskType].ProcessingTimes = append(m.data[taskType].ProcessingTimes, duration)
		m.data[taskType].TotalProcessingTime += duration
		m.data[taskType].TasksCompleted++
		m.data[taskType].AverageProcessingTime = m.data[taskType].TotalProcessingTime /
			time.Duration(m.data[taskType].TasksCompleted)

		// Cleanup
		delete(m.taskStartTimes, taskID)
	} else {
		// If no start time, just increment completion counter
		m.data[taskType].TasksCompleted++
	}
}

// initMetricsIfNeeded initializes metrics for a task type if not exists
func (m *Metrics) initMetricsIfNeeded(taskType string) *MetricsData {
	if _, exists := m.data[taskType]; !exists {
		m.data[taskType] = &MetricsData{
			ProcessingTimes: make([]time.Duration, 0),
		}
	}
	return m.data[taskType]
}

// RecordTaskError updates metrics for a failed task
func (m *Metrics) RecordTaskError(taskType string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := m.initMetricsIfNeeded(taskType)
	metrics.TasksFailed++
	metrics.LastError = err
	metrics.LastErrorTime = time.Now()
}

// RecordRoutingSuccess updates metrics for successful task routing
func (m *Metrics) RecordRoutingSuccess(taskType string, agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := m.initMetricsIfNeeded(taskType)
	metrics.RoutingSuccesses++
}

// RecordRoutingFailure updates metrics for failed task routing
func (m *Metrics) RecordRoutingFailure(taskType string, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := m.initMetricsIfNeeded(taskType)
	metrics.RoutingFailures++
}

// GetMetrics returns the current metrics for a task type
func (m *Metrics) GetMetrics(taskType string) *MetricsData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if metrics, exists := m.data[taskType]; exists {
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

// GetAllMetrics returns metrics for all task types
func (m *Metrics) GetAllMetrics() map[string]*MetricsData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a deep copy to avoid external modifications
	result := make(map[string]*MetricsData, len(m.data))
	for taskType, metrics := range m.data {
		metricsCopy := *metrics
		if metrics.ProcessingTimes != nil {
			metricsCopy.ProcessingTimes = make([]time.Duration, len(metrics.ProcessingTimes))
			copy(metricsCopy.ProcessingTimes, metrics.ProcessingTimes)
		}
		result[taskType] = &metricsCopy
	}
	return result
}

// ResetMetrics clears all metrics data
func (m *Metrics) ResetMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]*MetricsData)
	m.taskStartTimes = make(map[string]time.Time)
}
