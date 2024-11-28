package engine

import (
    "time"
    "encoding/json"
)


type SchemaDefinition struct {
    Type       string                 `json:"type"`
    Properties map[string]interface{} `json:"properties"`
    Required   []string              `json:"required"`
}

type RateLimit struct {
    RequestsPerSecond int     `json:"requestsPerSecond"`
    BurstSize        int     `json:"burstSize"`
}

type RetryPolicy struct {
    MaxAttempts     int           `json:"maxAttempts"`
    BackoffInitial  time.Duration `json:"backoffInitial"`
    BackoffMax      time.Duration `json:"backoffMax"`
    BackoffFactor   float64       `json:"backoffFactor"`
}

type ResourceQuota struct {
    MaxConcurrent    int    `json:"maxConcurrent"`
    MemoryLimit      string `json:"memoryLimit"`  // e.g., "512Mi"
    CPULimit         string `json:"cpuLimit"`     // e.g., "0.5"
}

type TaskConstraints struct {
    MaxInputLength     int      `json:"maxInputLength"`
    MaxOutputLength    int      `json:"maxOutputLength"`
    SupportedLanguages []string `json:"supportedLanguages"`
    ContentFilters    []string `json:"contentFilters"`
}

type TaskExample struct {
    Input  map[string]interface{} `json:"input"`
    Output map[string]interface{} `json:"output"`
}

type PerformanceSpec struct {
    AverageLatency string `json:"averageLatency"`
    Throughput     string `json:"throughput"`
}

// Core Types Improvements
type AITaskDefinition struct {
    TaskID          string            `json:"taskId"`
    Name            string            `json:"name"`
    Description     string            `json:"description"`
    Version         string            `json:"version"`
    InputSchema     SchemaDefinition  `json:"inputSchema"`
    OutputSchema    SchemaDefinition  `json:"outputSchema"`
    Capabilities    []Capability      `json:"capabilities"`
    Constraints     TaskConstraints   `json:"constraints"`
    Performance     PerformanceSpec   `json:"performance"`
    Examples        []TaskExample     `json:"examples"`
    Tags            []string          `json:"tags"`
    // Added fields for better production readiness
    Owner           string            `json:"owner"`
    Status          string            `json:"status"`  // active, deprecated, beta
    Timeout         time.Duration     `json:"timeout"`
    RetryPolicy     RetryPolicy      `json:"retryPolicy"`
    RateLimit       *RateLimit       `json:"rateLimit,omitempty"`
    ResourceQuota   *ResourceQuota   `json:"resourceQuota,omitempty"`
}

type TaskStatus struct {
    Phase     string `json:"phase"` // pending, running, completed, failed
    Message   string `json:"message,omitempty"`
    Progress  int    `json:"progress"`
    AttemptID string `json:"attemptId,omitempty"`
}

// Task Runtime Types
type TaskInstance struct {
    ID            string                 `json:"id"`
    DefinitionID  string                 `json:"definitionId"`
    Input         map[string]interface{} `json:"input"`
    Status        TaskStatus             `json:"status"`
    CreatedAt     time.Time             `json:"createdAt"`
    StartedAt     *time.Time            `json:"startedAt,omitempty"`
    CompletedAt   *time.Time            `json:"completedAt,omitempty"`
    Result        *TaskResult            `json:"result,omitempty"`
}

type ExecutionMetrics struct {
    ProcessingTime time.Duration `json:"processingTime"`
    InputSize      int          `json:"inputSize"`
    OutputSize     int          `json:"outputSize"`
}

type TaskResult struct {
    Output   map[string]interface{} `json:"output"`
    Metrics  ExecutionMetrics      `json:"metrics"`
    Error    string                `json:"error,omitempty"`
}
