package collector

import (
	"sync"
	"time"
)

type MetricsCollector struct {
    mu           sync.RWMutex
    metrics      map[string]*Metric
    labels       map[string]string
    lastUpdated  time.Time
}

type Metric struct {
    Name   string
    Value  float64
    Type   MetricType  // counter, gauge, histogram
    Labels map[string]string
}

type MetricType string

const (
    Counter   MetricType = "counter"
    Gauge     MetricType = "gauge"
    Histogram MetricType = "histogram"
)

func NewMetricsCollector(labels map[string]string) *MetricsCollector {
    return &MetricsCollector{
        metrics: make(map[string]*Metric),
        labels:  labels,
    }
}

func (mc *MetricsCollector) Record(name string, value float64, metricType MetricType) {
    mc.mu.Lock()
    defer mc.mu.Unlock()

    mc.metrics[name] = &Metric{
        Name:   name,
        Value:  value,
        Type:   metricType,
        Labels: mc.labels,
    }
    mc.lastUpdated = time.Now()
}

func (mc *MetricsCollector) Collect() map[string]*Metric {
    mc.mu.RLock()
    defer mc.mu.RUnlock()
    
    // Return a copy of metrics
    result := make(map[string]*Metric, len(mc.metrics))
    for k, v := range mc.metrics {
        result[k] = v
    }
    return result
}