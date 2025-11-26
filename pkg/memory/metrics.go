// Package memory provides observability and monitoring for memory implementations.
// It integrates OpenTelemetry for metrics and tracing as the default observability solution.
package memory

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the metric instruments for memory operations.
type Metrics struct {
	meter             metric.Meter
	loadDuration      metric.Float64Histogram
	saveDuration      metric.Float64Histogram
	clearDuration     metric.Float64Histogram
	operationCounter  metric.Int64Counter
	errorCounter      metric.Int64Counter
	memorySizeGauge   metric.Int64UpDownCounter
	activeMemoryGauge metric.Int64UpDownCounter
}

// NewMetrics creates a new Metrics instance with proper OTEL integration.
func NewMetrics(meter metric.Meter) *Metrics {
	loadDuration, _ := meter.Float64Histogram(
		"memory_load_duration_seconds",
		metric.WithDescription("Duration of memory load operations"),
		metric.WithUnit("s"),
	)
	saveDuration, _ := meter.Float64Histogram(
		"memory_save_duration_seconds",
		metric.WithDescription("Duration of memory save operations"),
		metric.WithUnit("s"),
	)
	clearDuration, _ := meter.Float64Histogram(
		"memory_clear_duration_seconds",
		metric.WithDescription("Duration of memory clear operations"),
		metric.WithUnit("s"),
	)
	operationCounter, _ := meter.Int64Counter(
		"memory_operations_total",
		metric.WithDescription("Total number of memory operations"),
	)
	errorCounter, _ := meter.Int64Counter(
		"memory_errors_total",
		metric.WithDescription("Total number of memory errors"),
	)
	memorySizeGauge, _ := meter.Int64UpDownCounter(
		"memory_size_current",
		metric.WithDescription("Current size of memory content"),
	)
	activeMemoryGauge, _ := meter.Int64UpDownCounter(
		"memory_instances_active",
		metric.WithDescription("Number of active memory instances"),
	)

	return &Metrics{
		meter:             meter,
		loadDuration:      loadDuration,
		saveDuration:      saveDuration,
		clearDuration:     clearDuration,
		operationCounter:  operationCounter,
		errorCounter:      errorCounter,
		memorySizeGauge:   memorySizeGauge,
		activeMemoryGauge: activeMemoryGauge,
	}
}

// RecordOperation records a memory operation with its attributes.
func (m *Metrics) RecordOperation(ctx context.Context, operation string, memoryType MemoryType, success bool) {
	if m == nil || m.operationCounter == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.String("memory_type", string(memoryType)),
		attribute.Bool("success", success),
	}
	m.operationCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordOperationDuration records the duration of a memory operation.
func (m *Metrics) RecordOperationDuration(ctx context.Context, operation string, memoryType MemoryType, duration time.Duration) {
	if m == nil {
		return
	}
	var histogram metric.Float64Histogram
	switch operation {
	case "load":
		histogram = m.loadDuration
	case "save":
		histogram = m.saveDuration
	case "clear":
		histogram = m.clearDuration
	default:
		return
	}

	if histogram == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("memory_type", string(memoryType)),
	}
	histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordError records a memory error.
func (m *Metrics) RecordError(ctx context.Context, operation string, memoryType MemoryType, errorType string) {
	if m == nil || m.errorCounter == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.String("memory_type", string(memoryType)),
		attribute.String("error_type", errorType),
	}
	m.errorCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordMemorySize records the size/length of memory content.
func (m *Metrics) RecordMemorySize(ctx context.Context, memoryType MemoryType, size int) {
	if m == nil || m.memorySizeGauge == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("memory_type", string(memoryType)),
	}
	m.memorySizeGauge.Add(ctx, int64(size), metric.WithAttributes(attrs...))
}

// RecordActiveMemory records the creation or deletion of a memory instance.
func (m *Metrics) RecordActiveMemory(ctx context.Context, memoryType MemoryType, delta int64) {
	if m == nil || m.activeMemoryGauge == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("memory_type", string(memoryType)),
	}
	m.activeMemoryGauge.Add(ctx, delta, metric.WithAttributes(attrs...))
}

// Tracer provides tracing functionality for memory operations.
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer creates a new Tracer instance.
func NewTracer() *Tracer {
	return &Tracer{
		tracer: otel.Tracer("github.com/lookatitude/beluga-ai/pkg/memory"),
	}
}

// StartSpan starts a new span for a memory operation.
//
//nolint:spancheck // Spans are intentionally returned for caller to manage lifecycle
func (t *Tracer) StartSpan(ctx context.Context, operation string, memoryType MemoryType, memoryKey string) (context.Context, trace.Span) {
	if t == nil || t.tracer == nil {
		return ctx, nil
	}
	ctx, span := t.tracer.Start(ctx, "memory."+operation)
	return ctx, span
}

// RecordSpanError records an error on the span.
func (t *Tracer) RecordSpanError(span trace.Span, err error) {
	if t == nil || span == nil {
		return
	}
	if err != nil {
		span.RecordError(err)
	}
}

// Global metrics and tracer instances.
var (
	globalMetrics *Metrics
	globalTracer  *Tracer
)

// SetGlobalMetrics sets the global metrics instance with proper OTEL meter.
func SetGlobalMetrics(meter metric.Meter) {
	globalMetrics = NewMetrics(meter)
}

// GetGlobalMetrics returns the global metrics instance.
func GetGlobalMetrics() *Metrics {
	return globalMetrics
}

// SetGlobalTracer sets the global tracer instance.
func SetGlobalTracer() {
	globalTracer = NewTracer()
}

// GetGlobalTracer returns the global tracer instance.
func GetGlobalTracer() *Tracer {
	return globalTracer
}

// Logger provides structured logging for memory operations.
// It integrates with OpenTelemetry tracing for consistent log correlation.
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new structured logger for memory operations.
func NewLogger(logger *slog.Logger) *Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return &Logger{
		logger: logger,
	}
}

// LogMemoryOperation logs memory operations with context.
func (l *Logger) LogMemoryOperation(ctx context.Context, level slog.Level, operation string, memoryType MemoryType, memoryKey string, messageCount int, duration time.Duration, err error) {
	if l == nil || l.logger == nil {
		return
	}
	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.String("memory_type", string(memoryType)),
		slog.String("memory_key", memoryKey),
		slog.Int("message_count", messageCount),
		slog.Duration("duration", duration),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		l.logger.LogAttrs(ctx, level, "Memory operation completed with error", attrs...)
	} else {
		l.logger.LogAttrs(ctx, level, "Memory operation completed successfully", attrs...)
	}
}

// LogMemoryLifecycle logs memory lifecycle events.
func (l *Logger) LogMemoryLifecycle(ctx context.Context, event string, memoryType MemoryType, memoryKey string, additionalAttrs ...slog.Attr) {
	if l == nil || l.logger == nil {
		return
	}
	attrs := []slog.Attr{
		slog.String("event", event),
		slog.String("memory_type", string(memoryType)),
		slog.String("memory_key", memoryKey),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add any additional attributes
	attrs = append(attrs, additionalAttrs...)

	l.logger.LogAttrs(ctx, slog.LevelInfo, "Memory lifecycle event", attrs...)
}

// LogError logs errors with context.
func (l *Logger) LogError(ctx context.Context, err error, operation string, memoryType MemoryType, memoryKey string, additionalAttrs ...slog.Attr) {
	if l == nil || l.logger == nil {
		return
	}
	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.String("memory_type", string(memoryType)),
		slog.String("memory_key", memoryKey),
		slog.String("error", err.Error()),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add any additional attributes
	attrs = append(attrs, additionalAttrs...)

	l.logger.LogAttrs(ctx, slog.LevelError, "Memory operation failed", attrs...)
}

// Global logger instance.
var globalLogger *Logger

// SetGlobalLogger sets the global logger instance.
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance.
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		globalLogger = NewLogger(nil)
	}
	return globalLogger
}

// Default convenience functions that use the global logger.
var (
	LogMemoryOperation = func(ctx context.Context, level slog.Level, operation string, memoryType MemoryType, memoryKey string, messageCount int, duration time.Duration, err error) {
		GetGlobalLogger().LogMemoryOperation(ctx, level, operation, memoryType, memoryKey, messageCount, duration, err)
	}

	LogMemoryLifecycle = func(ctx context.Context, event string, memoryType MemoryType, memoryKey string, additionalAttrs ...slog.Attr) {
		GetGlobalLogger().LogMemoryLifecycle(ctx, event, memoryType, memoryKey, additionalAttrs...)
	}

	LogError = func(ctx context.Context, err error, operation string, memoryType MemoryType, memoryKey string, additionalAttrs ...slog.Attr) {
		GetGlobalLogger().LogError(ctx, err, operation, memoryType, memoryKey, additionalAttrs...)
	}
)
