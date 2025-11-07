package config

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds the metrics for the config package.
type Metrics struct {
	configLoadsTotal      metric.Int64Counter
	configLoadDuration    metric.Float64Histogram
	configErrorsTotal     metric.Int64Counter
	validationDuration    metric.Float64Histogram
	validationErrorsTotal metric.Int64Counter
}

// NewMetrics creates a new metrics instance for the config package.
func NewMetrics(meter metric.Meter) (*Metrics, error) {
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

	return &Metrics{
		configLoadsTotal:      configLoadsTotal,
		configLoadDuration:    configLoadDuration,
		configErrorsTotal:     configErrorsTotal,
		validationDuration:    validationDuration,
		validationErrorsTotal: validationErrorsTotal,
	}, nil
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{}
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

// Global metrics instance - initialized lazily.
var (
	globalMetrics              *Metrics
	globalMetricsExplicitlySet bool
)

// GetGlobalMetrics returns the global metrics instance, creating it if necessary.
func GetGlobalMetrics() *Metrics {
	if globalMetrics == nil && !globalMetricsExplicitlySet {
		if metrics, err := NewMetrics(otel.Meter("github.com/lookatitude/beluga-ai/pkg/config")); err == nil {
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
func SetGlobalMetrics(m *Metrics) {
	globalMetrics = m
	globalMetricsExplicitlySet = true // Always mark as explicitly set, even if nil
}
