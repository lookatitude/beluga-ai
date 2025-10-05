// Package contracts defines the API contracts for Config package OpenTelemetry integration.
// These interfaces enable comprehensive observability with metrics, tracing, and logging
// while maintaining constitutional compliance and performance requirements.
package contracts

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ConfigMetrics defines the interface for comprehensive configuration observability.
// It provides standardized metrics collection, distributed tracing, and structured logging
// for all configuration operations with minimal performance overhead requirement.
type ConfigMetrics interface {
	// RecordOperation records metrics for a configuration operation.
	// Required by constitution for all OTEL implementations.
	RecordOperation(ctx context.Context, operation string, duration time.Duration, success bool) error

	// RecordLoad records metrics for configuration loading operations.
	RecordLoad(ctx context.Context, provider, format string, duration time.Duration, success bool) error

	// RecordValidation records metrics for configuration validation operations.
	RecordValidation(ctx context.Context, validationType string, duration time.Duration, success bool, errorCount int) error

	// RecordProviderOperation records metrics for provider-specific operations.
	RecordProviderOperation(ctx context.Context, provider, operation string, duration time.Duration, success bool) error

	// StartSpan creates a distributed tracing span for configuration operations.
	StartSpan(ctx context.Context, operation, provider, format string) (context.Context, trace.Span)

	// GetHealthMetrics returns current health and performance metrics.
	GetHealthMetrics() ConfigHealthMetrics
}

// ConfigMetricsFactory defines the interface for creating metrics instances.
type ConfigMetricsFactory interface {
	// NewMetrics creates a new metrics instance with OTEL meter and tracer.
	// Required constitutional implementation.
	NewMetrics(meter metric.Meter, tracer trace.Tracer) (ConfigMetrics, error)

	// NoOpMetrics returns a no-op metrics implementation for testing.
	// Required by constitution for development scenarios.
	NoOpMetrics() ConfigMetrics
}

// ConfigHealthMetrics provides current health and performance metrics.
type ConfigHealthMetrics struct {
	// Load Metrics
	TotalLoads      int64   `json:"total_loads"`
	SuccessfulLoads int64   `json:"successful_loads"`
	FailedLoads     int64   `json:"failed_loads"`
	SuccessRate     float64 `json:"success_rate"`

	// Performance Metrics
	AverageLoadTime time.Duration `json:"average_load_time"`
	P95LoadTime     time.Duration `json:"p95_load_time"`
	P99LoadTime     time.Duration `json:"p99_load_time"`
	LoadThroughput  float64       `json:"load_throughput"` // loads/second

	// Validation Metrics
	TotalValidations      int64         `json:"total_validations"`
	ValidationErrors      int64         `json:"validation_errors"`
	ValidationSuccessRate float64       `json:"validation_success_rate"`
	AverageValidationTime time.Duration `json:"average_validation_time"`

	// Provider Metrics
	ProviderHealth  map[string]ProviderHealthInfo `json:"provider_health"`
	ActiveProviders int                           `json:"active_providers"`
	CacheHitRate    float64                       `json:"cache_hit_rate"`

	// Error Metrics
	ErrorRates  map[string]float64 `json:"error_rates"`  // by error code
	ErrorCounts map[string]int64   `json:"error_counts"` // by error code

	// Collection Metadata
	LastUpdated      time.Time     `json:"last_updated"`
	CollectionPeriod time.Duration `json:"collection_period"`
}

// ProviderHealthInfo contains health information for a specific provider.
type ProviderHealthInfo struct {
	Status              string        `json:"status"`
	Availability        float64       `json:"availability"` // 0.0 to 1.0
	AverageResponseTime time.Duration `json:"average_response_time"`
	ErrorRate           float64       `json:"error_rate"` // 0.0 to 1.0
	LastHealthCheck     time.Time     `json:"last_health_check"`
	LoadsCompleted      int64         `json:"loads_completed"`
	ConsecutiveErrors   int           `json:"consecutive_errors"`
	SupportedFormats    []string      `json:"supported_formats"`
	Capabilities        []string      `json:"capabilities"`
}

// OperationTracker defines the interface for tracking individual configuration operations.
type OperationTracker interface {
	// StartOperation begins tracking a configuration operation.
	StartOperation(ctx context.Context, op OperationInfo) (context.Context, OperationHandle)

	// CompleteOperation marks an operation as completed with success/failure status.
	CompleteOperation(handle OperationHandle, result OperationResult) error

	// GetActiveOperations returns information about currently active operations.
	GetActiveOperations() []ActiveOperationInfo

	// GetOperationHistory returns recent operation history for analysis.
	GetOperationHistory(limit int) []CompletedOperationInfo
}

// OperationInfo contains information about a configuration operation being tracked.
type OperationInfo struct {
	OperationID   string            `json:"operation_id"`
	OperationType string            `json:"operation_type"` // load, validate, reload, etc.
	Provider      string            `json:"provider"`
	Format        string            `json:"format,omitempty"`
	ConfigPath    string            `json:"config_path,omitempty"`
	StartTime     time.Time         `json:"start_time"`
	Context       map[string]string `json:"context,omitempty"`
}

// OperationHandle represents a handle to a tracked operation.
type OperationHandle interface {
	// GetOperationID returns the unique operation identifier.
	GetOperationID() string

	// AddAttribute adds custom attributes to the operation tracking.
	AddAttribute(key string, value interface{}) error

	// SetStatus updates the operation status.
	SetStatus(status string) error
}

// OperationResult contains the result information for a completed operation.
type OperationResult struct {
	Success          bool              `json:"success"`
	Duration         time.Duration     `json:"duration"`
	ErrorCode        string            `json:"error_code,omitempty"`
	ErrorMessage     string            `json:"error_message,omitempty"`
	ValidationTime   time.Duration     `json:"validation_time,omitempty"`
	ValidationErrors int               `json:"validation_errors"`
	ConfigSize       int64             `json:"config_size,omitempty"`
	SourcesParsed    int               `json:"sources_parsed"`
	CacheHit         bool              `json:"cache_hit"`
	Attributes       map[string]string `json:"attributes,omitempty"`
	EndTime          time.Time         `json:"end_time"`
}

// ActiveOperationInfo provides information about currently running operations.
type ActiveOperationInfo struct {
	OperationInfo
	ElapsedTime   time.Duration `json:"elapsed_time"`
	CurrentStatus string        `json:"current_status"`
	Progress      float64       `json:"progress,omitempty"` // 0.0 to 1.0
}

// CompletedOperationInfo provides information about completed operations.
type CompletedOperationInfo struct {
	OperationInfo
	OperationResult
}

// MetricsCollector defines the interface for collecting and aggregating metrics data.
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

// Standard attributes used across all configuration metrics and traces.
var StandardAttributes = struct {
	// Operation Attributes
	OperationType string
	OperationName string

	// Provider Attributes
	Provider string
	Format   string

	// Source Attributes
	ConfigPath string
	EnvPrefix  string
	SourceType string

	// Result Attributes
	Success   string
	ErrorCode string
	LoadTime  string

	// Performance Attributes
	CacheHit       string
	SourceCount    string
	ValidationTime string
}{
	OperationType:  "config.operation.type",
	OperationName:  "config.operation.name",
	Provider:       "config.provider",
	Format:         "config.format",
	ConfigPath:     "config.path",
	EnvPrefix:      "config.env_prefix",
	SourceType:     "config.source.type",
	Success:        "config.success",
	ErrorCode:      "config.error.code",
	LoadTime:       "config.load_time",
	CacheHit:       "config.cache.hit",
	SourceCount:    "config.sources.count",
	ValidationTime: "config.validation.time",
}

// ObservabilityConfiguration combines all observability configuration options.
type ObservabilityConfiguration struct {
	// Metrics Configuration
	EnableMetrics   bool          `json:"enable_metrics"`
	MetricsPrefix   string        `json:"metrics_prefix"`
	MetricsInterval time.Duration `json:"metrics_interval"`

	// Tracing Configuration
	EnableTracing   bool    `json:"enable_tracing"`
	TracingSampling float64 `json:"tracing_sampling"`
	TracingService  string  `json:"tracing_service"`

	// Logging Configuration
	EnableLogging     bool   `json:"enable_logging"`
	LogLevel          string `json:"log_level"`
	StructuredLogging bool   `json:"structured_logging"`

	// Health Monitoring
	EnableHealthCheck bool          `json:"enable_health_check"`
	HealthInterval    time.Duration `json:"health_interval"`

	// Performance Settings
	MaxOverheadPercent float64 `json:"max_overhead_percent"` // Default: 5.0
}
