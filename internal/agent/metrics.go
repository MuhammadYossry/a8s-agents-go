package agent

import (
    "sync"
    "time"

    "github.com/Relax-N-Tax/AgentNexus/internal/task/types"
    "github.com/Relax-N-Tax/AgentNexus/pkg/metrics"
)

// AgentMetrics tracks performance metrics for an individual AI agent
type AgentMetrics struct {
    agentID        string
    collector      *metrics.MetricsCollector
    mu            sync.RWMutex
    
    // Performance metrics
    taskLatencies  *TimeWindow    // Rolling window of task latencies
    successRate    *RateCounter   // Success rate calculator
    throughput     *RateCounter   // Tasks processed per minute
    
    // Resource utilization
    memoryUsage    float64
    cpuUsage       float64
    queueUtilization float64
    
    // Task-specific metrics
    taskCounts     map[string]int64    // Counts by task type
    errorCounts    map[string]int64    // Counts by error type
    
    // Time-based metrics
    lastUpdated    time.Time
    uptime        time.Duration
    startTime     time.Time
}

// TimeWindow maintains a rolling window of durations
type TimeWindow struct {
    mu       sync.RWMutex
    window   []time.Duration
    capacity int
    index    int
}

// RateCounter calculates rates over time
type RateCounter struct {
    mu       sync.RWMutex
    success  int64
    total    int64
    window   time.Duration
}

func NewAgentMetrics(agentID string, windowSize int) *AgentMetrics {
    return &AgentMetrics{
        agentID:     agentID,
        collector:   metrics.NewMetricsCollector(map[string]string{
            "agent_id": agentID,
        }),
        taskLatencies: NewTimeWindow(windowSize),
        successRate:   NewRateCounter(5 * time.Minute),  // 5-minute window
        throughput:    NewRateCounter(1 * time.Minute),  // 1-minute window
        taskCounts:    make(map[string]int64),
        errorCounts:   make(map[string]int64),
        startTime:     time.Now(),
    }
}

func (am *AgentMetrics) RecordTaskExecution(task *task.TaskInstance, duration time.Duration, err error) {
    am.mu.Lock()
    defer am.mu.Unlock()

    // Update basic metrics
    am.taskLatencies.Add(duration)
    am.throughput.Increment(err == nil)
    am.successRate.Increment(err == nil)
    
    // Update task counts
    am.taskCounts[task.DefinitionID]++
    if err != nil {
        am.errorCounts[err.Error()]++
    }

    // Record to metrics collector
    am.collector.Record("task_duration_ms", float64(duration.Milliseconds()), metrics.Histogram)
    am.collector.Record("tasks_processed_total", float64(am.throughput.Total()), metrics.Counter)
    am.collector.Record("task_success_rate", am.successRate.Rate(), metrics.Gauge)
    
    am.lastUpdated = time.Now()
}

func (am *AgentMetrics) UpdateResourceMetrics(memUsage, cpuUsage, queueUtil float64) {
    am.mu.Lock()
    defer am.mu.Unlock()

    am.memoryUsage = memUsage
    am.cpuUsage = cpuUsage
    am.queueUtilization = queueUtil

    // Record resource metrics
    am.collector.Record("memory_usage_percent", memUsage, metrics.Gauge)
    am.collector.Record("cpu_usage_percent", cpuUsage, metrics.Gauge)
    am.collector.Record("queue_utilization", queueUtil, metrics.Gauge)
}

func (am *AgentMetrics) GetPerformanceStats() PerformanceStats {
    am.mu.RLock()
    defer am.mu.RUnlock()

    return PerformanceStats{
        AverageLatency:    am.taskLatencies.Average(),
        SuccessRate:       am.successRate.Rate(),
        TasksPerMinute:    am.throughput.Rate(),
        MemoryUsage:       am.memoryUsage,
        CPUUsage:          am.cpuUsage,
        QueueUtilization:  am.queueUtilization,
        Uptime:           time.Since(am.startTime),
        TaskCounts:        am.copyTaskCounts(),
        ErrorCounts:       am.copyErrorCounts(),
    }
}

// TimeWindow implementation
func NewTimeWindow(capacity int) *TimeWindow {
    return &TimeWindow{
        window:   make([]time.Duration, capacity),
        capacity: capacity,
    }
}

func (tw *TimeWindow) Add(duration time.Duration) {
    tw.mu.Lock()
    defer tw.mu.Unlock()
    
    tw.window[tw.index] = duration
    tw.index = (tw.index + 1) % tw.capacity
}

func (tw *TimeWindow) Average() time.Duration {
    tw.mu.RLock()
    defer tw.mu.RUnlock()
    
    var sum time.Duration
    var count int
    for _, d := range tw.window {
        if d > 0 {
            sum += d
            count++
        }
    }
    
    if count == 0 {
        return 0
    }
    return sum / time.Duration(count)
}

// RateCounter implementation
func NewRateCounter(window time.Duration) *RateCounter {
    return &RateCounter{
        window: window,
    }
}

func (rc *RateCounter) Increment(success bool) {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    
    rc.total++
    if success {
        rc.success++
    }
}

func (rc *RateCounter) Rate() float64 {
    rc.mu.RLock()
    defer rc.mu.RUnlock()
    
    if rc.total == 0 {
        return 0
    }
    return float64(rc.success) / float64(rc.total)
}

func (rc *RateCounter) Total() int64 {
    rc.mu.RLock()
    defer rc.mu.RUnlock()
    return rc.total
}

// Helper methods
func (am *AgentMetrics) copyTaskCounts() map[string]int64 {
    counts := make(map[string]int64)
    for k, v := range am.taskCounts {
        counts[k] = v
    }
    return counts
}

func (am *AgentMetrics) copyErrorCounts() map[string]int64 {
    counts := make(map[string]int64)
    for k, v := range am.errorCounts {
        counts[k] = v
    }
    return counts
}

type PerformanceStats struct {
    AverageLatency    time.Duration     `json:"averageLatency"`
    SuccessRate       float64           `json:"successRate"`
    TasksPerMinute    float64           `json:"tasksPerMinute"`
    MemoryUsage       float64           `json:"memoryUsage"`
    CPUUsage          float64           `json:"cpuUsage"`
    QueueUtilization  float64           `json:"queueUtilization"`
    Uptime           time.Duration      `json:"uptime"`
    TaskCounts        map[string]int64  `json:"taskCounts"`
    ErrorCounts       map[string]int64  `json:"errorCounts"`
}