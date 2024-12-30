// metrics/metrics.go
package metrics

import (
	"strings"
	"sync"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

type Metrics struct {
	mu             sync.RWMutex
	data           map[types.MetricsKey]*types.MetricsData
	taskStartTimes map[string]time.Time
}

func NewMetrics() types.MetricsCollector {
	return &Metrics{
		data:           make(map[types.MetricsKey]*types.MetricsData),
		taskStartTimes: make(map[string]time.Time),
	}
}

func createMetricsKey(req types.TaskRequirement) types.MetricsKey {
	return types.MetricsKey{
		SkillPath: strings.Join(req.SkillPath, "."),
		Action:    req.Action,
	}
}

func (m *Metrics) RecordTaskStart(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskStartTimes[taskID] = time.Now()
}

func (m *Metrics) RecordTaskComplete(requirements types.TaskRequirement, taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := createMetricsKey(requirements)
	if _, exists := m.data[key]; !exists {
		m.data[key] = &types.MetricsData{
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

func (m *Metrics) initMetricsIfNeeded(key types.MetricsKey) *types.MetricsData {
	if _, exists := m.data[key]; !exists {
		m.data[key] = &types.MetricsData{
			ProcessingTimes: make([]time.Duration, 0),
		}
	}
	return m.data[key]
}

func (m *Metrics) RecordTaskError(requirements types.TaskRequirement, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := createMetricsKey(requirements)
	metrics := m.initMetricsIfNeeded(key)
	metrics.TasksFailed++
	metrics.LastError = err
	metrics.LastErrorTime = time.Now()
}

func (m *Metrics) RecordRoutingSuccess(requirements types.TaskRequirement, agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := createMetricsKey(requirements)
	metrics := m.initMetricsIfNeeded(key)
	metrics.RoutingSuccesses++
}

func (m *Metrics) GetMetrics(requirements types.TaskRequirement) *types.MetricsData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := createMetricsKey(requirements)
	if metrics, exists := m.data[key]; exists {
		metricsCopy := *metrics
		if metrics.ProcessingTimes != nil {
			metricsCopy.ProcessingTimes = make([]time.Duration, len(metrics.ProcessingTimes))
			copy(metricsCopy.ProcessingTimes, metrics.ProcessingTimes)
		}
		return &metricsCopy
	}
	return nil
}

func (m *Metrics) RecordRoutingFailure(requirements types.TaskRequirement, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := createMetricsKey(requirements)
	metrics := m.initMetricsIfNeeded(key)
	metrics.RoutingFailures++
}

func (m *Metrics) GetAllMetrics() map[types.MetricsKey]*types.MetricsData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[types.MetricsKey]*types.MetricsData, len(m.data))
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

func (m *Metrics) ResetMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[types.MetricsKey]*types.MetricsData)
	m.taskStartTimes = make(map[string]time.Time)
}
