// Package monitoring provides package-specific metrics definitions
package monitoring

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// PackageMetrics defines comprehensive metrics for the monitoring package.
type PackageMetrics struct {
	// System metrics
	monitorStartTime time.Time
	uptime           metric.Float64ObservableGauge
	memoryUsage      metric.Int64ObservableGauge
	goroutines       metric.Int64ObservableGauge

	// Request and operation metrics
	totalRequests     metric.Int64Counter
	activeOperations  metric.Int64UpDownCounter
	operationDuration metric.Float64Histogram
	requestsPerSecond metric.Float64ObservableGauge

	// Component-specific metrics
	healthCheckDuration metric.Float64Histogram
	safetyCheckDuration metric.Float64Histogram
	ethicsCheckDuration metric.Float64Histogram
	tracingDuration     metric.Float64Histogram
	loggingDuration     metric.Float64Histogram

	// Error metrics
	totalErrors       metric.Int64Counter
	safetyErrors      metric.Int64Counter
	ethicsErrors      metric.Int64Counter
	healthCheckErrors metric.Int64Counter
	tracingErrors     metric.Int64Counter
	loggingErrors     metric.Int64Counter

	// Component health metrics
	loggerHealth        metric.Int64Gauge
	tracerHealth        metric.Int64Gauge
	metricsHealth       metric.Int64Gauge
	healthCheckerHealth metric.Int64Gauge
	safetyCheckerHealth metric.Int64Gauge
	ethicsCheckerHealth metric.Int64Gauge

	// Framework-wide metrics
	apiRequests          metric.Int64Counter
	databaseQueries      metric.Int64Counter
	cacheHits            metric.Int64Counter
	cacheMisses          metric.Int64Counter
	externalAPICalls     metric.Int64Counter
	validationOperations metric.Int64Counter

	// Performance metrics
	responseTime metric.Float64Histogram
	throughput   metric.Float64ObservableGauge
	errorRate    metric.Float64ObservableGauge
	availability metric.Float64ObservableGauge

	// Resource metrics
	cpuUsage  metric.Float64ObservableGauge
	diskUsage metric.Float64ObservableGauge
	networkIO metric.Float64ObservableGauge

	// Custom business metrics
	customCounters   map[string]metric.Int64Counter
	customGauges     map[string]metric.Float64ObservableGauge
	customHistograms map[string]metric.Float64Histogram

	// Internal fields
	meter metric.Meter
}

// NewPackageMetrics creates a new comprehensive package metrics instance.
func NewPackageMetrics(meter metric.Meter) *PackageMetrics {
	pm := &PackageMetrics{
		monitorStartTime: time.Now(),
		meter:            meter,
		customCounters:   make(map[string]metric.Int64Counter),
		customGauges:     make(map[string]metric.Float64ObservableGauge),
		customHistograms: make(map[string]metric.Float64Histogram),
	}

	var err error

	// Initialize system metrics
	pm.uptime, err = meter.Float64ObservableGauge(
		"monitoring_uptime_seconds",
		metric.WithDescription("Monitoring system uptime in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	pm.memoryUsage, err = meter.Int64ObservableGauge(
		"monitoring_memory_usage_bytes",
		metric.WithDescription("Memory usage of the monitoring system"),
		metric.WithUnit("By"),
	)
	if err != nil {
		panic(err)
	}

	pm.goroutines, err = meter.Int64ObservableGauge(
		"monitoring_goroutines_total",
		metric.WithDescription("Number of goroutines used by monitoring system"),
	)
	if err != nil {
		panic(err)
	}

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

	pm.tracingDuration, err = meter.Float64Histogram(
		"monitoring_tracing_duration_seconds",
		metric.WithDescription("Duration of tracing operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	pm.loggingDuration, err = meter.Float64Histogram(
		"monitoring_logging_duration_seconds",
		metric.WithDescription("Duration of logging operations"),
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

	pm.tracingErrors, err = meter.Int64Counter(
		"monitoring_tracing_errors_total",
		metric.WithDescription("Total number of tracing errors"),
	)
	if err != nil {
		panic(err)
	}

	pm.loggingErrors, err = meter.Int64Counter(
		"monitoring_logging_errors_total",
		metric.WithDescription("Total number of logging errors"),
	)
	if err != nil {
		panic(err)
	}

	// Initialize framework-wide metrics
	pm.apiRequests, err = meter.Int64Counter(
		"monitoring_api_requests_total",
		metric.WithDescription("Total number of API requests"),
	)
	if err != nil {
		panic(err)
	}

	pm.databaseQueries, err = meter.Int64Counter(
		"monitoring_database_queries_total",
		metric.WithDescription("Total number of database queries"),
	)
	if err != nil {
		panic(err)
	}

	pm.cacheHits, err = meter.Int64Counter(
		"monitoring_cache_hits_total",
		metric.WithDescription("Total number of cache hits"),
	)
	if err != nil {
		panic(err)
	}

	pm.cacheMisses, err = meter.Int64Counter(
		"monitoring_cache_misses_total",
		metric.WithDescription("Total number of cache misses"),
	)
	if err != nil {
		panic(err)
	}

	pm.externalAPICalls, err = meter.Int64Counter(
		"monitoring_external_api_calls_total",
		metric.WithDescription("Total number of external API calls"),
	)
	if err != nil {
		panic(err)
	}

	pm.validationOperations, err = meter.Int64Counter(
		"monitoring_validation_operations_total",
		metric.WithDescription("Total number of validation operations"),
	)
	if err != nil {
		panic(err)
	}

	// Initialize performance metrics
	pm.responseTime, err = meter.Float64Histogram(
		"monitoring_response_time_seconds",
		metric.WithDescription("Response time for operations"),
		metric.WithUnit("s"),
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

// RecordRequest records a monitoring request.
func (pm *PackageMetrics) RecordRequest(ctx context.Context, operation string) {
	pm.totalRequests.Add(ctx, 1,
		metric.WithAttributes(attribute.String("operation", operation)))
}

// RecordOperationStart records the start of an operation.
func (pm *PackageMetrics) RecordOperationStart(ctx context.Context, operation string) {
	pm.activeOperations.Add(ctx, 1,
		metric.WithAttributes(attribute.String("operation", operation)))
}

// RecordOperationEnd records the end of an operation.
func (pm *PackageMetrics) RecordOperationEnd(ctx context.Context, operation string, duration time.Duration) {
	pm.activeOperations.Add(ctx, -1,
		metric.WithAttributes(attribute.String("operation", operation)))

	pm.operationDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(attribute.String("operation", operation)))
}

// RecordHealthCheck records a health check operation.
func (pm *PackageMetrics) RecordHealthCheck(ctx context.Context, component string, duration time.Duration, success bool) {
	if success {
		pm.healthCheckDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(attribute.String("component", component)))
	} else {
		pm.healthCheckErrors.Add(ctx, 1,
			metric.WithAttributes(attribute.String("component", component)))
	}
}

// RecordSafetyCheck records a safety check operation.
func (pm *PackageMetrics) RecordSafetyCheck(ctx context.Context, duration time.Duration, success bool) {
	if success {
		pm.safetyCheckDuration.Record(ctx, duration.Seconds())
	} else {
		pm.safetyErrors.Add(ctx, 1)
	}
}

// RecordEthicsCheck records an ethics check operation.
func (pm *PackageMetrics) RecordEthicsCheck(ctx context.Context, duration time.Duration, success bool) {
	if success {
		pm.ethicsCheckDuration.Record(ctx, duration.Seconds())
	} else {
		pm.ethicsErrors.Add(ctx, 1)
	}
}

// UpdateComponentHealth updates the health status of a component.
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

// RecordError records a general monitoring error.
func (pm *PackageMetrics) RecordError(ctx context.Context, component string, err error) {
	pm.totalErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("component", component),
			attribute.String("error_type", err.Error()),
		))
}

// RecordTracingOperation records a tracing operation.
func (pm *PackageMetrics) RecordTracingOperation(ctx context.Context, duration time.Duration, success bool) {
	if success {
		pm.tracingDuration.Record(ctx, duration.Seconds())
	} else {
		pm.tracingErrors.Add(ctx, 1)
	}
}

// RecordLoggingOperation records a logging operation.
func (pm *PackageMetrics) RecordLoggingOperation(ctx context.Context, duration time.Duration, success bool) {
	if success {
		pm.loggingDuration.Record(ctx, duration.Seconds())
	} else {
		pm.loggingErrors.Add(ctx, 1)
	}
}

// RecordAPIRequest records an API request.
func (pm *PackageMetrics) RecordAPIRequest(ctx context.Context, method, endpoint string) {
	pm.apiRequests.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("endpoint", endpoint),
		))
}

// RecordDatabaseQuery records a database query.
func (pm *PackageMetrics) RecordDatabaseQuery(ctx context.Context, operation, table string, duration time.Duration) {
	pm.databaseQueries.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("table", table),
		))
	pm.responseTime.Record(ctx, duration.Seconds(),
		metric.WithAttributes(attribute.String("type", "database")))
}

// RecordCacheOperation records a cache operation.
func (pm *PackageMetrics) RecordCacheOperation(ctx context.Context, operation, key string, hit bool) {
	if hit {
		pm.cacheHits.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("key", key),
			))
	} else {
		pm.cacheMisses.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("key", key),
			))
	}
}

// RecordExternalAPICall records an external API call.
func (pm *PackageMetrics) RecordExternalAPICall(ctx context.Context, service, endpoint string, duration time.Duration, success bool) {
	pm.externalAPICalls.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("service", service),
			attribute.String("endpoint", endpoint),
			attribute.Bool("success", success),
		))
	pm.responseTime.Record(ctx, duration.Seconds(),
		metric.WithAttributes(attribute.String("type", "external_api")))
}

// RecordValidationOperation records a validation operation.
func (pm *PackageMetrics) RecordValidationOperation(ctx context.Context, validatorType string, duration time.Duration, success bool) {
	pm.validationOperations.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("validator_type", validatorType),
			attribute.Bool("success", success),
		))
}

// RegisterCustomCounter registers a custom counter metric.
func (pm *PackageMetrics) RegisterCustomCounter(name, description string) error {
	counter, err := pm.getMeter().Int64Counter(name, metric.WithDescription(description))
	if err != nil {
		return err
	}
	pm.customCounters[name] = counter
	return nil
}

// RegisterCustomHistogram registers a custom histogram metric.
func (pm *PackageMetrics) RegisterCustomHistogram(name, description string) error {
	histogram, err := pm.getMeter().Float64Histogram(name, metric.WithDescription(description))
	if err != nil {
		return err
	}
	pm.customHistograms[name] = histogram
	return nil
}

// RecordCustomCounter records a value for a custom counter.
func (pm *PackageMetrics) RecordCustomCounter(ctx context.Context, name string, value int64, labels map[string]string) {
	if counter, exists := pm.customCounters[name]; exists {
		attrs := make([]attribute.KeyValue, 0, len(labels))
		for k, v := range labels {
			attrs = append(attrs, attribute.String(k, v))
		}
		counter.Add(ctx, value, metric.WithAttributes(attrs...))
	}
}

// RecordCustomHistogram records a value for a custom histogram.
func (pm *PackageMetrics) RecordCustomHistogram(ctx context.Context, name string, value float64, labels map[string]string) {
	if histogram, exists := pm.customHistograms[name]; exists {
		attrs := make([]attribute.KeyValue, 0, len(labels))
		for k, v := range labels {
			attrs = append(attrs, attribute.String(k, v))
		}
		histogram.Record(ctx, value, metric.WithAttributes(attrs...))
	}
}

// GetUptime returns the uptime of the monitoring system.
func (pm *PackageMetrics) GetUptime() time.Duration {
	return time.Since(pm.monitorStartTime)
}

// getMeter returns the OpenTelemetry meter.
func (pm *PackageMetrics) getMeter() metric.Meter {
	return pm.meter
}
