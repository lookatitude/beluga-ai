package config

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the metrics for the config package
// Implements the ConfigMetrics interface for constitutional compliance
type Metrics struct {
	// Core metrics
	configLoadsTotal      metric.Int64Counter
	configLoadDuration    metric.Float64Histogram
	configErrorsTotal     metric.Int64Counter
	validationDuration    metric.Float64Histogram
	validationErrorsTotal metric.Int64Counter

	// Extended metrics for constitutional compliance
	loadCounter         metric.Int64Counter
	loadDuration        metric.Float64Histogram
	errorCounter        metric.Int64Counter
	validationCounter   metric.Int64Counter
	providerHealthGauge metric.Int64Gauge

	// OTEL components
	meter  metric.Meter
	tracer trace.Tracer
}

// ConfigHealthMetrics provides current health and performance metrics.
// Required by constitution for comprehensive observability.
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

// NewMetrics creates a new metrics instance for the config package
// Implements the ConfigMetricsFactory interface for constitutional compliance
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	// Get tracer
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/config")

	// Core metrics
	configLoadsTotal, err := meter.Int64Counter(
		"config_loads_total",
		metric.WithDescription("Total number of configuration loads"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	configLoadDuration, err := meter.Float64Histogram(
		"config_load_duration_seconds",
		metric.WithDescription("Duration of configuration loading operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	configErrorsTotal, err := meter.Int64Counter(
		"config_errors_total",
		metric.WithDescription("Total number of configuration errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	validationDuration, err := meter.Float64Histogram(
		"config_validation_duration_seconds",
		metric.WithDescription("Duration of configuration validation operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	validationErrorsTotal, err := meter.Int64Counter(
		"config_validation_errors_total",
		metric.WithDescription("Total number of configuration validation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Extended metrics for constitutional compliance
	loadCounter, err := meter.Int64Counter(
		"config_operations_total",
		metric.WithDescription("Total number of configuration operations by type"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	loadDuration, err := meter.Float64Histogram(
		"config_operation_duration_seconds",
		metric.WithDescription("Duration of configuration operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	errorCounter, err := meter.Int64Counter(
		"config_operation_errors_total",
		metric.WithDescription("Total number of configuration operation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	validationCounter, err := meter.Int64Counter(
		"config_validations_total",
		metric.WithDescription("Total number of configuration validations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	providerHealthGauge, err := meter.Int64Gauge(
		"config_provider_health",
		metric.WithDescription("Current health status of configuration providers"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		configLoadsTotal:      configLoadsTotal,
		configLoadDuration:    configLoadDuration,
		configErrorsTotal:     configErrorsTotal,
		validationDuration:    validationDuration,
		validationErrorsTotal: validationErrorsTotal,
		loadCounter:           loadCounter,
		loadDuration:          loadDuration,
		errorCounter:          errorCounter,
		validationCounter:     validationCounter,
		providerHealthGauge:   providerHealthGauge,
		meter:                 meter,
		tracer:                tracer,
	}, nil
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{}
}

// RecordConfigLoad records a configuration load operation
func (m *Metrics) RecordConfigLoad(ctx context.Context, duration time.Duration, success bool, source string) {
	if m == nil {
		return
	}

	if m.configLoadsTotal != nil {
		m.configLoadsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.Bool("success", success),
				attribute.String("source", source),
			))
	}
	if m.configLoadDuration != nil {
		m.configLoadDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("source", source),
			))
	}
	if !success && m.configErrorsTotal != nil {
		m.configErrorsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", "load"),
				attribute.String("source", source),
			))
	}
}

// RecordValidation records a configuration validation operation (legacy signature)
// This method is kept for backward compatibility but delegates to the extended version
func (m *Metrics) RecordValidation(ctx context.Context, duration time.Duration, success bool) {
	// Delegate to extended version with default parameters
	m.RecordValidationExtended(ctx, "general", duration, success, 0)
}

// RecordOperation records metrics for a configuration operation.
// Required by constitution for all OTEL implementations.
func (m *Metrics) RecordOperation(ctx context.Context, operation string, duration time.Duration, success bool) error {
	if m == nil {
		return nil
	}

	if m.loadCounter != nil {
		m.loadCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.Bool("success", success),
			))
	}
	if m.loadDuration != nil {
		m.loadDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("operation", operation),
			))
	}
	if !success && m.errorCounter != nil {
		m.errorCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
			))
	}
	return nil
}

// RecordLoad records metrics for configuration loading operations.
func (m *Metrics) RecordLoad(ctx context.Context, provider, format string, duration time.Duration, success bool) error {
	return m.RecordOperation(ctx, "load", duration, success)
}

// RecordValidationExtended records metrics for configuration validation operations.
// Extended version with validation type and error count for constitutional compliance.
func (m *Metrics) RecordValidationExtended(ctx context.Context, validationType string, duration time.Duration, success bool, errorCount int) error {
	if m == nil {
		return nil
	}

	if m.validationCounter != nil {
		m.validationCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("type", validationType),
				attribute.Bool("success", success),
				attribute.Int("errors", errorCount),
			))
	}
	if m.validationDuration != nil {
		m.validationDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("type", validationType),
			))
	}
	return nil
}

// RecordProviderOperation records metrics for provider-specific operations.
func (m *Metrics) RecordProviderOperation(ctx context.Context, provider, operation string, duration time.Duration, success bool) error {
	if m == nil {
		return nil
	}

	if m.loadCounter != nil {
		m.loadCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("operation", operation),
				attribute.Bool("success", success),
			))
	}
	if m.loadDuration != nil {
		m.loadDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("operation", operation),
			))
	}
	return nil
}

// StartSpan creates a distributed tracing span for configuration operations.
func (m *Metrics) StartSpan(ctx context.Context, operation, provider, format string) (context.Context, trace.Span) {
	if m == nil || m.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := m.tracer.Start(ctx, operation,
		trace.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("format", format),
		))
	return ctx, span
}

// GetHealthMetrics returns current health and performance metrics.
// Required by constitution for observability.
func (m *Metrics) GetHealthMetrics() ConfigHealthMetrics {
	if m == nil {
		return ConfigHealthMetrics{}
	}

	// This is a simplified implementation - in a real implementation,
	// these metrics would be collected from actual metric values
	return ConfigHealthMetrics{
		TotalLoads:            0, // Would be collected from actual metrics
		SuccessfulLoads:       0,
		FailedLoads:           0,
		SuccessRate:           1.0,
		AverageLoadTime:       0,
		P95LoadTime:           0,
		P99LoadTime:           0,
		LoadThroughput:        0,
		TotalValidations:      0,
		ValidationErrors:      0,
		ValidationSuccessRate: 1.0,
		AverageValidationTime: 0,
		ProviderHealth:        nil,
		ActiveProviders:       0,
		CacheHitRate:          0,
		ErrorRates:            nil,
		ErrorCounts:           nil,
		LastUpdated:           time.Now(),
		CollectionPeriod:      time.Minute,
	}
}

// Global metrics instance - initialized lazily
var globalMetrics *Metrics

// GetGlobalMetrics returns the global metrics instance, creating it if necessary
func GetGlobalMetrics() *Metrics {
	if globalMetrics == nil {
		if metrics, err := NewMetrics(otel.Meter("github.com/lookatitude/beluga-ai/pkg/config")); err == nil {
			globalMetrics = metrics
		} else {
			// If metrics creation fails, use no-op metrics
			globalMetrics = NoOpMetrics()
		}
	}
	return globalMetrics
}

// SetGlobalMetrics allows setting a custom metrics instance for testing
func SetGlobalMetrics(m *Metrics) {
	globalMetrics = m
}
