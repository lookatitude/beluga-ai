// Package contracts defines the API contracts for ChatModels package OpenTelemetry integration.
// These interfaces enable comprehensive observability with metrics, tracing, and logging
// while maintaining constitutional compliance and performance requirements.
package contracts

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ChatModelMetrics defines the interface for comprehensive chat model observability.
// It provides standardized metrics collection, distributed tracing, and structured logging
// for all chat model operations with <5% performance overhead requirement.
type ChatModelMetrics interface {
	// RecordOperation records metrics for a chat model operation.
	// Required by constitution for all OTEL implementations.
	RecordOperation(ctx context.Context, operation string, duration time.Duration, success bool) error

	// StartSpan creates a distributed tracing span for chat model operations.
	// Returns context with span and the span for additional attributes.
	StartSpan(ctx context.Context, operation, provider, model string) (context.Context, trace.Span)

	// RecordError records error metrics with structured error information.
	RecordError(ctx context.Context, operation, provider, errorCode string) error

	// RecordTokenUsage records token consumption metrics for cost tracking.
	RecordTokenUsage(ctx context.Context, provider, model string, inputTokens, outputTokens int) error

	// RecordLatency records operation latency for performance monitoring.
	RecordLatency(ctx context.Context, operation, provider, model string, duration time.Duration) error

	// RecordThroughput records throughput metrics for capacity planning.
	RecordThroughput(ctx context.Context, provider, model string, requestsPerSecond float64) error

	// GetHealthMetrics returns current health and performance metrics.
	GetHealthMetrics() HealthMetrics
}

// MetricsFactory defines the interface for creating metrics instances.
// Supports both standard and no-op implementations for testing.
type MetricsFactory interface {
	// NewMetrics creates a new metrics instance with OTEL meter and tracer.
	// Required constitutional implementation.
	NewMetrics(meter metric.Meter, tracer trace.Tracer) (ChatModelMetrics, error)

	// NoOpMetrics returns a no-op metrics implementation for testing.
	// Required by constitution for development scenarios.
	NoOpMetrics() ChatModelMetrics
}

// OperationTracker defines the interface for tracking individual operations
// with detailed context and correlation information.
type OperationTracker interface {
	// StartOperation begins tracking a chat model operation.
	StartOperation(ctx context.Context, op OperationInfo) (context.Context, OperationHandle)

	// CompleteOperation marks an operation as completed with success/failure status.
	CompleteOperation(handle OperationHandle, result OperationResult) error

	// GetActiveOperations returns information about currently active operations.
	GetActiveOperations() []ActiveOperationInfo

	// GetOperationHistory returns recent operation history for analysis.
	GetOperationHistory(limit int) []CompletedOperationInfo
}

// OperationInfo contains information about a chat model operation being tracked.
type OperationInfo struct {
	OperationID   string            `json:"operation_id"`
	OperationType string            `json:"operation_type"` // generate, stream, batch, etc.
	ProviderName  string            `json:"provider_name"`
	ModelName     string            `json:"model_name"`
	UserID        string            `json:"user_id,omitempty"`
	SessionID     string            `json:"session_id,omitempty"`
	RequestID     string            `json:"request_id,omitempty"`
	StartTime     time.Time         `json:"start_time"`
	Context       map[string]string `json:"context,omitempty"`
}

// OperationHandle represents a handle to a tracked operation for completion reporting.
type OperationHandle interface {
	// GetOperationID returns the unique operation identifier.
	GetOperationID() string

	// AddAttribute adds custom attributes to the operation tracking.
	AddAttribute(key string, value interface{}) error

	// SetStatus updates the operation status (in-progress, completed, failed).
	SetStatus(status string) error
}

// OperationResult contains the result information for a completed operation.
type OperationResult struct {
	Success      bool              `json:"success"`
	Duration     time.Duration     `json:"duration"`
	TokensUsed   TokenUsage        `json:"tokens_used,omitempty"`
	ErrorCode    string            `json:"error_code,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	ResponseSize int               `json:"response_size,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	EndTime      time.Time         `json:"end_time"`
}

// TokenUsage represents token consumption information for cost tracking.
type TokenUsage struct {
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	CostUSD          float64 `json:"cost_usd,omitempty"`
	CachedTokens     int     `json:"cached_tokens,omitempty"`
	PromptTokens     int     `json:"prompt_tokens,omitempty"`     // Provider-specific
	CompletionTokens int     `json:"completion_tokens,omitempty"` // Provider-specific
}

// ActiveOperationInfo provides information about currently running operations.
type ActiveOperationInfo struct {
	OperationInfo
	ElapsedTime   time.Duration `json:"elapsed_time"`
	CurrentStatus string        `json:"current_status"`
	Progress      float64       `json:"progress,omitempty"` // 0.0 to 1.0 for streaming
}

// CompletedOperationInfo provides information about completed operations.
type CompletedOperationInfo struct {
	OperationInfo
	OperationResult
	Metrics OperationMetrics `json:"metrics"`
}

// OperationMetrics contains detailed metrics for a completed operation.
type OperationMetrics struct {
	RequestLatency   time.Duration `json:"request_latency"`
	ProcessingTime   time.Duration `json:"processing_time"`
	NetworkLatency   time.Duration `json:"network_latency,omitempty"`
	QueueTime        time.Duration `json:"queue_time,omitempty"`
	BytesTransferred int64         `json:"bytes_transferred"`
	RetryCount       int           `json:"retry_count"`
	CacheHit         bool          `json:"cache_hit"`
}

// HealthMetrics provides current health and performance metrics.
type HealthMetrics struct {
	// Request Metrics
	TotalRequests   int64   `json:"total_requests"`
	SuccessRequests int64   `json:"success_requests"`
	FailureRequests int64   `json:"failure_requests"`
	SuccessRate     float64 `json:"success_rate"`

	// Performance Metrics
	AverageLatency time.Duration `json:"average_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`
	ThroughputRPS  float64       `json:"throughput_rps"`

	// Resource Metrics
	ActiveOperations int   `json:"active_operations"`
	QueuedOperations int   `json:"queued_operations"`
	MemoryUsage      int64 `json:"memory_usage"`

	// Error Metrics
	ErrorRates  map[string]float64 `json:"error_rates"`  // by error code
	ErrorCounts map[string]int64   `json:"error_counts"` // by error code

	// Provider Metrics
	ProviderHealth map[string]ProviderHealthInfo `json:"provider_health"`

	// Collection Metadata
	LastUpdated      time.Time     `json:"last_updated"`
	CollectionPeriod time.Duration `json:"collection_period"`
}

// ProviderHealthInfo contains health information for a specific provider.
type ProviderHealthInfo struct {
	Status            string        `json:"status"`
	Availability      float64       `json:"availability"` // 0.0 to 1.0
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorRate         float64       `json:"error_rate"` // 0.0 to 1.0
	LastHealthCheck   time.Time     `json:"last_health_check"`
	ConsecutiveErrors int           `json:"consecutive_errors"`
}

// MetricsCollector defines the interface for collecting and aggregating
// metrics data from multiple sources.
type MetricsCollector interface {
	// StartCollection begins metrics collection with the specified interval.
	StartCollection(interval time.Duration) error

	// StopCollection stops metrics collection and cleanup resources.
	StopCollection() error

	// GetCollectionStatus returns the current status of metrics collection.
	GetCollectionStatus() CollectionStatus

	// FlushMetrics forces immediate flush of collected metrics.
	FlushMetrics() error
}

// CollectionStatus represents the current state of metrics collection.
type CollectionStatus struct {
	Active             bool          `json:"active"`
	CollectionInterval time.Duration `json:"collection_interval"`
	LastCollection     time.Time     `json:"last_collection"`
	MetricsCollected   int64         `json:"metrics_collected"`
	CollectionErrors   int64         `json:"collection_errors"`
	StartTime          time.Time     `json:"start_time"`
}

// TracingConfiguration defines configuration for distributed tracing.
type TracingConfiguration struct {
	// Tracing Settings
	EnableTracing     bool    `json:"enable_tracing"`
	SamplingRate      float64 `json:"sampling_rate"` // 0.0 to 1.0
	MaxSpanAttributes int     `json:"max_span_attributes"`

	// Span Configuration
	RecordExceptions bool     `json:"record_exceptions"`
	RecordStackTrace bool     `json:"record_stack_trace"`
	AttributeFilters []string `json:"attribute_filters,omitempty"`

	// Export Configuration
	ExportTimeout time.Duration `json:"export_timeout"`
	BatchTimeout  time.Duration `json:"batch_timeout"`
	MaxBatchSize  int           `json:"max_batch_size"`
}

// MetricsConfiguration defines configuration for metrics collection.
type MetricsConfiguration struct {
	// Collection Settings
	EnableMetrics      bool          `json:"enable_metrics"`
	CollectionInterval time.Duration `json:"collection_interval"`
	MetricPrefix       string        `json:"metric_prefix"`

	// Aggregation Settings
	HistogramBuckets []float64 `json:"histogram_buckets"`
	EnableHistograms bool      `json:"enable_histograms"`
	EnableCounters   bool      `json:"enable_counters"`
	EnableGauges     bool      `json:"enable_gauges"`

	// Export Configuration
	ExportInterval    time.Duration `json:"export_interval"`
	MaxMetricsInBatch int           `json:"max_metrics_in_batch"`
	EnableCompression bool          `json:"enable_compression"`
}

// LoggingConfiguration defines configuration for structured logging.
type LoggingConfiguration struct {
	// Logging Settings
	EnableLogging    bool   `json:"enable_logging"`
	LogLevel         string `json:"log_level"`  // debug, info, warn, error
	LogFormat        string `json:"log_format"` // json, text
	EnableStructured bool   `json:"enable_structured"`

	// Context Settings
	IncludeTraceID   bool     `json:"include_trace_id"`
	IncludeSpanID    bool     `json:"include_span_id"`
	AdditionalFields []string `json:"additional_fields,omitempty"`

	// Output Configuration
	OutputDestination string `json:"output_destination"` // stdout, file, syslog
	LogFilePath       string `json:"log_file_path,omitempty"`
	MaxFileSize       int64  `json:"max_file_size,omitempty"`
	MaxBackups        int    `json:"max_backups,omitempty"`
}

// ObservabilityConfig combines all observability configuration options.
type ObservabilityConfig struct {
	Tracing TracingConfiguration `json:"tracing"`
	Metrics MetricsConfiguration `json:"metrics"`
	Logging LoggingConfiguration `json:"logging"`

	// Global Settings
	ServiceName      string            `json:"service_name"`
	ServiceVersion   string            `json:"service_version"`
	Environment      string            `json:"environment"`
	GlobalAttributes map[string]string `json:"global_attributes,omitempty"`

	// Performance Settings
	MaxOverheadPercent          float64 `json:"max_overhead_percent"` // Default: 5.0
	EnablePerformanceMonitoring bool    `json:"enable_performance_monitoring"`
}

// StandardAttributes defines the standard attributes used across all metrics and traces.
var StandardAttributes = struct {
	// Operation Attributes
	OperationType string
	OperationName string

	// Provider Attributes
	ProviderName string
	ModelName    string

	// Request Attributes
	RequestID string
	UserID    string
	SessionID string

	// Response Attributes
	StatusCode string
	ErrorCode  string
	TokenCount string

	// Performance Attributes
	Duration   string
	RetryCount string
	CacheHit   string
}{
	OperationType: "chatmodels.operation.type",
	OperationName: "chatmodels.operation.name",
	ProviderName:  "chatmodels.provider.name",
	ModelName:     "chatmodels.model.name",
	RequestID:     "chatmodels.request.id",
	UserID:        "chatmodels.user.id",
	SessionID:     "chatmodels.session.id",
	StatusCode:    "chatmodels.response.status",
	ErrorCode:     "chatmodels.error.code",
	TokenCount:    "chatmodels.tokens.count",
	Duration:      "chatmodels.duration",
	RetryCount:    "chatmodels.retry.count",
	CacheHit:      "chatmodels.cache.hit",
}
