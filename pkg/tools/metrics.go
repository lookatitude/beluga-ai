// Package tools provides OTEL metrics for tool execution.
package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the metrics for the tools package.
type Metrics struct {
	// Execution metrics
	executionsTotal    metric.Int64Counter
	executionDuration  metric.Float64Histogram
	executionsInFlight metric.Int64UpDownCounter

	// Batch metrics
	batchesTotal  metric.Int64Counter
	batchSize     metric.Int64Histogram
	batchDuration metric.Float64Histogram

	// Error metrics
	errorsTotal metric.Int64Counter

	// Registry metrics
	registeredTools metric.Int64UpDownCounter

	// Tracer
	tracer trace.Tracer
}

// NewMetrics creates a new metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	var err error
	m := &Metrics{}

	// Execution metrics
	m.executionsTotal, err = meter.Int64Counter(
		"tools_executions_total",
		metric.WithDescription("Total number of tool executions"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executionsTotal metric: %w", err)
	}

	m.executionDuration, err = meter.Float64Histogram(
		"tools_execution_duration_seconds",
		metric.WithDescription("Duration of tool executions in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executionDuration metric: %w", err)
	}

	m.executionsInFlight, err = meter.Int64UpDownCounter(
		"tools_executions_in_flight",
		metric.WithDescription("Number of tool executions currently in flight"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executionsInFlight metric: %w", err)
	}

	// Batch metrics
	m.batchesTotal, err = meter.Int64Counter(
		"tools_batches_total",
		metric.WithDescription("Total number of batch executions"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create batchesTotal metric: %w", err)
	}

	m.batchSize, err = meter.Int64Histogram(
		"tools_batch_size",
		metric.WithDescription("Size of batch executions"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create batchSize metric: %w", err)
	}

	m.batchDuration, err = meter.Float64Histogram(
		"tools_batch_duration_seconds",
		metric.WithDescription("Duration of batch executions in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create batchDuration metric: %w", err)
	}

	// Error metrics
	m.errorsTotal, err = meter.Int64Counter(
		"tools_errors_total",
		metric.WithDescription("Total number of tool errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create errorsTotal metric: %w", err)
	}

	// Registry metrics
	m.registeredTools, err = meter.Int64UpDownCounter(
		"tools_registered_total",
		metric.WithDescription("Number of registered tools"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create registeredTools metric: %w", err)
	}

	if tracer == nil {
		tracer = trace.NewNoopTracerProvider().Tracer("tools")
	}
	m.tracer = tracer

	return m, nil
}

// RecordExecution records a tool execution.
func (m *Metrics) RecordExecution(ctx context.Context, toolName, toolType string, duration time.Duration, success bool) {
	if m == nil {
		return
	}

	status := "success"
	if !success {
		status = "failure"
	}

	attrs := metric.WithAttributes(
		attribute.String("tool_name", toolName),
		attribute.String("tool_type", toolType),
		attribute.String("status", status),
	)

	if m.executionsTotal != nil {
		m.executionsTotal.Add(ctx, 1, attrs)
	}
	if m.executionDuration != nil {
		m.executionDuration.Record(ctx, duration.Seconds(), attrs)
	}
}

// StartExecution increments the in-flight counter.
func (m *Metrics) StartExecution(ctx context.Context, toolName, toolType string) {
	if m == nil || m.executionsInFlight == nil {
		return
	}
	m.executionsInFlight.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("tool_name", toolName),
			attribute.String("tool_type", toolType),
		))
}

// EndExecution decrements the in-flight counter.
func (m *Metrics) EndExecution(ctx context.Context, toolName, toolType string) {
	if m == nil || m.executionsInFlight == nil {
		return
	}
	m.executionsInFlight.Add(ctx, -1,
		metric.WithAttributes(
			attribute.String("tool_name", toolName),
			attribute.String("tool_type", toolType),
		))
}

// RecordBatch records a batch execution.
func (m *Metrics) RecordBatch(ctx context.Context, toolName, toolType string, batchSize int, duration time.Duration, success bool) {
	if m == nil {
		return
	}

	status := "success"
	if !success {
		status = "failure"
	}

	attrs := metric.WithAttributes(
		attribute.String("tool_name", toolName),
		attribute.String("tool_type", toolType),
		attribute.String("status", status),
	)

	if m.batchesTotal != nil {
		m.batchesTotal.Add(ctx, 1, attrs)
	}
	if m.batchSize != nil {
		m.batchSize.Record(ctx, int64(batchSize), attrs)
	}
	if m.batchDuration != nil {
		m.batchDuration.Record(ctx, duration.Seconds(), attrs)
	}
}

// RecordError records a tool error.
func (m *Metrics) RecordError(ctx context.Context, toolName, toolType, errorCode string) {
	if m == nil || m.errorsTotal == nil {
		return
	}
	m.errorsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("tool_name", toolName),
			attribute.String("tool_type", toolType),
			attribute.String("error_code", errorCode),
		))
}

// RecordToolRegistered records a tool being registered.
func (m *Metrics) RecordToolRegistered(ctx context.Context, toolName, toolType string) {
	if m == nil || m.registeredTools == nil {
		return
	}
	m.registeredTools.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("tool_name", toolName),
			attribute.String("tool_type", toolType),
		))
}

// RecordToolUnregistered records a tool being unregistered.
func (m *Metrics) RecordToolUnregistered(ctx context.Context, toolName, toolType string) {
	if m == nil || m.registeredTools == nil {
		return
	}
	m.registeredTools.Add(ctx, -1,
		metric.WithAttributes(
			attribute.String("tool_name", toolName),
			attribute.String("tool_type", toolType),
		))
}

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("tools"),
	}
}

// InitMetrics initializes the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if tracer == nil {
			tracer = trace.NewNoopTracerProvider().Tracer("tools")
		}
		metrics, err := NewMetrics(meter, tracer)
		if err != nil {
			// If metrics creation fails, use no-op metrics
			globalMetrics = NoOpMetrics()
			return
		}
		globalMetrics = metrics
	})
}

// GetMetrics returns the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func GetMetrics() *Metrics {
	return globalMetrics
}
