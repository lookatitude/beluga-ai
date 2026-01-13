package config

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the metrics for the config package.
type Metrics struct {
	configLoadsTotal      metric.Int64Counter
	configLoadDuration    metric.Float64Histogram
	configErrorsTotal     metric.Int64Counter
	validationDuration    metric.Float64Histogram
	validationErrorsTotal metric.Int64Counter
	tracer                trace.Tracer
}

// NewMetrics creates a new metrics instance for the config package.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
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

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/config")
	}

	return &Metrics{
		configLoadsTotal:      configLoadsTotal,
		configLoadDuration:    configLoadDuration,
		configErrorsTotal:     configErrorsTotal,
		validationDuration:    validationDuration,
		validationErrorsTotal: validationErrorsTotal,
		tracer:                tracer,
	}, nil
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("config"),
	}
}

// RecordConfigLoad records a configuration load operation.
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

// RecordValidation records a configuration validation operation.
func (m *Metrics) RecordValidation(ctx context.Context, duration time.Duration, success bool) {
	if m == nil {
		return
	}

	if m.validationDuration != nil {
		m.validationDuration.Record(ctx, duration.Seconds())
	}
	if !success && m.validationErrorsTotal != nil {
		m.validationErrorsTotal.Add(ctx, 1)
	}
}

// Global metrics instance - initialized once.
var (
	globalMetrics              *Metrics
	globalMetricsExplicitlySet bool
	metricsOnce                sync.Once
)

// InitMetrics initializes the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if tracer == nil {
			tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/config")
		}
		metrics, err := NewMetrics(meter, tracer)
		if err != nil {
			// If metrics creation fails, use no-op metrics
			globalMetrics = NoOpMetrics()
			return
		}
		globalMetrics = metrics
		globalMetricsExplicitlySet = true
	})
}

// GetMetrics returns the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func GetMetrics() *Metrics {
	return globalMetrics
}

// GetGlobalMetrics returns the global metrics instance, creating it if necessary.
// Deprecated: Use InitMetrics(meter, tracer) and GetMetrics() instead for consistency.
func GetGlobalMetrics() *Metrics {
	if globalMetrics == nil && !globalMetricsExplicitlySet {
		meter := otel.Meter("github.com/lookatitude/beluga-ai/pkg/config")
		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/config")
		if metrics, err := NewMetrics(meter, tracer); err == nil {
			globalMetrics = metrics
		} else {
			// If metrics creation fails, use no-op metrics
			globalMetrics = NoOpMetrics()
		}
	}
	// If explicitly set to nil, return nil (for testing)
	// Otherwise, if still nil, return no-op metrics (shouldn't happen, but safety check)
	if globalMetrics == nil && !globalMetricsExplicitlySet {
		globalMetrics = NoOpMetrics()
	}
	return globalMetrics
}

// SetGlobalMetrics allows setting a custom metrics instance for testing.
// Deprecated: Use InitMetrics(meter) instead for consistency.
func SetGlobalMetrics(m *Metrics) {
	globalMetrics = m
	globalMetricsExplicitlySet = true // Always mark as explicitly set, even if nil
}
