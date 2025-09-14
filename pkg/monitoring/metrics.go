// Package monitoring provides package-specific metrics definitions
package monitoring

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// PackageMetrics defines metrics specific to the monitoring package
type PackageMetrics struct {
	// Monitoring system metrics
	monitorStartTime    time.Time
	totalRequests       metric.Int64Counter
	activeOperations    metric.Int64UpDownCounter
	operationDuration   metric.Float64Histogram
	healthCheckDuration metric.Float64Histogram
	safetyCheckDuration metric.Float64Histogram
	ethicsCheckDuration metric.Float64Histogram

	// Error metrics
	totalErrors       metric.Int64Counter
	safetyErrors      metric.Int64Counter
	ethicsErrors      metric.Int64Counter
	healthCheckErrors metric.Int64Counter

	// Component health metrics
	loggerHealth        metric.Int64Gauge
	tracerHealth        metric.Int64Gauge
	metricsHealth       metric.Int64Gauge
	healthCheckerHealth metric.Int64Gauge
	safetyCheckerHealth metric.Int64Gauge
	ethicsCheckerHealth metric.Int64Gauge
}

// NewPackageMetrics creates a new package metrics instance
func NewPackageMetrics(meter metric.Meter) *PackageMetrics {
	pm := &PackageMetrics{
		monitorStartTime: time.Now(),
	}

	var err error

	// Initialize counters
	pm.totalRequests, err = meter.Int64Counter(
		"monitoring_requests_total",
		metric.WithDescription("Total number of monitoring requests"),
	)
	if err != nil {
		panic(err)
	}

	pm.activeOperations, err = meter.Int64UpDownCounter(
		"monitoring_active_operations",
		metric.WithDescription("Number of currently active monitoring operations"),
	)
	if err != nil {
		panic(err)
	}

	// Initialize histograms
	pm.operationDuration, err = meter.Float64Histogram(
		"monitoring_operation_duration_seconds",
		metric.WithDescription("Duration of monitoring operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	pm.healthCheckDuration, err = meter.Float64Histogram(
		"monitoring_health_check_duration_seconds",
		metric.WithDescription("Duration of health checks"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	pm.safetyCheckDuration, err = meter.Float64Histogram(
		"monitoring_safety_check_duration_seconds",
		metric.WithDescription("Duration of safety checks"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	pm.ethicsCheckDuration, err = meter.Float64Histogram(
		"monitoring_ethics_check_duration_seconds",
		metric.WithDescription("Duration of ethics checks"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	// Initialize error counters
	pm.totalErrors, err = meter.Int64Counter(
		"monitoring_errors_total",
		metric.WithDescription("Total number of monitoring errors"),
	)
	if err != nil {
		panic(err)
	}

	pm.safetyErrors, err = meter.Int64Counter(
		"monitoring_safety_errors_total",
		metric.WithDescription("Total number of safety check errors"),
	)
	if err != nil {
		panic(err)
	}

	pm.ethicsErrors, err = meter.Int64Counter(
		"monitoring_ethics_errors_total",
		metric.WithDescription("Total number of ethics check errors"),
	)
	if err != nil {
		panic(err)
	}

	pm.healthCheckErrors, err = meter.Int64Counter(
		"monitoring_health_check_errors_total",
		metric.WithDescription("Total number of health check errors"),
	)
	if err != nil {
		panic(err)
	}

	// Initialize health gauges
	pm.loggerHealth, err = meter.Int64Gauge(
		"monitoring_logger_health",
		metric.WithDescription("Logger component health (1=healthy, 0=unhealthy)"),
	)
	if err != nil {
		panic(err)
	}

	pm.tracerHealth, err = meter.Int64Gauge(
		"monitoring_tracer_health",
		metric.WithDescription("Tracer component health (1=healthy, 0=unhealthy)"),
	)
	if err != nil {
		panic(err)
	}

	pm.metricsHealth, err = meter.Int64Gauge(
		"monitoring_metrics_health",
		metric.WithDescription("Metrics component health (1=healthy, 0=unhealthy)"),
	)
	if err != nil {
		panic(err)
	}

	pm.healthCheckerHealth, err = meter.Int64Gauge(
		"monitoring_health_checker_health",
		metric.WithDescription("Health checker component health (1=healthy, 0=unhealthy)"),
	)
	if err != nil {
		panic(err)
	}

	pm.safetyCheckerHealth, err = meter.Int64Gauge(
		"monitoring_safety_checker_health",
		metric.WithDescription("Safety checker component health (1=healthy, 0=unhealthy)"),
	)
	if err != nil {
		panic(err)
	}

	pm.ethicsCheckerHealth, err = meter.Int64Gauge(
		"monitoring_ethics_checker_health",
		metric.WithDescription("Ethics checker component health (1=healthy, 0=unhealthy)"),
	)
	if err != nil {
		panic(err)
	}

	return pm
}

// RecordRequest records a monitoring request
func (pm *PackageMetrics) RecordRequest(ctx context.Context, operation string) {
	pm.totalRequests.Add(ctx, 1,
		metric.WithAttributes(attribute.String("operation", operation)))
}

// RecordOperationStart records the start of an operation
func (pm *PackageMetrics) RecordOperationStart(ctx context.Context, operation string) {
	pm.activeOperations.Add(ctx, 1,
		metric.WithAttributes(attribute.String("operation", operation)))
}

// RecordOperationEnd records the end of an operation
func (pm *PackageMetrics) RecordOperationEnd(ctx context.Context, operation string, duration time.Duration) {
	pm.activeOperations.Add(ctx, -1,
		metric.WithAttributes(attribute.String("operation", operation)))

	pm.operationDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(attribute.String("operation", operation)))
}

// RecordHealthCheck records a health check operation
func (pm *PackageMetrics) RecordHealthCheck(ctx context.Context, component string, duration time.Duration, success bool) {
	if success {
		pm.healthCheckDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(attribute.String("component", component)))
	} else {
		pm.healthCheckErrors.Add(ctx, 1,
			metric.WithAttributes(attribute.String("component", component)))
	}
}

// RecordSafetyCheck records a safety check operation
func (pm *PackageMetrics) RecordSafetyCheck(ctx context.Context, duration time.Duration, success bool) {
	if success {
		pm.safetyCheckDuration.Record(ctx, duration.Seconds())
	} else {
		pm.safetyErrors.Add(ctx, 1)
	}
}

// RecordEthicsCheck records an ethics check operation
func (pm *PackageMetrics) RecordEthicsCheck(ctx context.Context, duration time.Duration, success bool) {
	if success {
		pm.ethicsCheckDuration.Record(ctx, duration.Seconds())
	} else {
		pm.ethicsErrors.Add(ctx, 1)
	}
}

// UpdateComponentHealth updates the health status of a component
func (pm *PackageMetrics) UpdateComponentHealth(ctx context.Context, component string, healthy bool) {
	var value int64
	if healthy {
		value = 1
	}

	switch component {
	case "logger":
		pm.loggerHealth.Record(ctx, value)
	case "tracer":
		pm.tracerHealth.Record(ctx, value)
	case "metrics":
		pm.metricsHealth.Record(ctx, value)
	case "health_checker":
		pm.healthCheckerHealth.Record(ctx, value)
	case "safety_checker":
		pm.safetyCheckerHealth.Record(ctx, value)
	case "ethics_checker":
		pm.ethicsCheckerHealth.Record(ctx, value)
	}
}

// RecordError records a general monitoring error
func (pm *PackageMetrics) RecordError(ctx context.Context, component string, err error) {
	pm.totalErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("component", component),
			attribute.String("error_type", err.Error()),
		))
}

// GetUptime returns the uptime of the monitoring system
func (pm *PackageMetrics) GetUptime() time.Duration {
	return time.Since(pm.monitorStartTime)
}
