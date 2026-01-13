// Package core provides metrics definitions for the Beluga AI framework.
// It defines standard metrics for Runnable executions and other core operations.
package core

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the metric instruments for core operations.
type Metrics struct {
	// Runnable execution metrics
	runnableInvokes  metric.Int64Counter
	runnableBatches  metric.Int64Counter
	runnableStreams  metric.Int64Counter
	runnableErrors   metric.Int64Counter
	runnableDuration metric.Float64Histogram

	// Batch operation metrics
	batchSize     metric.Int64Histogram
	batchDuration metric.Float64Histogram

	// Stream operation metrics
	streamDuration metric.Float64Histogram
	streamChunks   metric.Int64Counter

	// Tracer for span creation
	tracer trace.Tracer
}

// createCounter creates a counter instrument with consistent error handling.
func createCounter(meter metric.Meter, name, desc, unit string) (metric.Int64Counter, error) {
	return meter.Int64Counter(name, metric.WithDescription(desc), metric.WithUnit(unit))
}

// createFloat64Histogram creates a float64 histogram with consistent error handling.
func createFloat64Histogram(meter metric.Meter, name, desc, unit string) (metric.Float64Histogram, error) {
	return meter.Float64Histogram(name, metric.WithDescription(desc), metric.WithUnit(unit))
}

// createInt64Histogram creates an int64 histogram with consistent error handling.
func createInt64Histogram(meter metric.Meter, name, desc, unit string) (metric.Int64Histogram, error) {
	return meter.Int64Histogram(name, metric.WithDescription(desc), metric.WithUnit(unit))
}

// createCounters creates all counter instruments.
func createCounters(meter metric.Meter) (metric.Int64Counter, metric.Int64Counter, metric.Int64Counter, metric.Int64Counter, error) {
	runnableInvokes, err := createCounter(meter, "runnable_invokes_total",
		"Total number of Runnable.Invoke calls", "1")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	runnableBatches, err := createCounter(meter, "runnable_batches_total",
		"Total number of Runnable.Batch calls", "1")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	runnableStreams, err := createCounter(meter, "runnable_streams_total",
		"Total number of Runnable.Stream calls", "1")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	runnableErrors, err := createCounter(meter, "runnable_errors_total",
		"Total number of Runnable execution errors", "1")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return runnableInvokes, runnableBatches, runnableStreams, runnableErrors, nil
}

// createHistograms creates all histogram instruments.
func createHistograms(meter metric.Meter) (metric.Float64Histogram, metric.Int64Histogram, metric.Float64Histogram, metric.Float64Histogram, error) {
	runnableDuration, err := createFloat64Histogram(meter, "runnable_duration_seconds",
		"Duration of Runnable operations", "s")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	batchSize, err := createInt64Histogram(meter, "runnable_batch_size",
		"Size of Runnable batch operations", "1")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	batchDuration, err := createFloat64Histogram(meter, "runnable_batch_duration_seconds",
		"Duration of Runnable batch operations", "s")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	streamDuration, err := createFloat64Histogram(meter, "runnable_stream_duration_seconds",
		"Duration of Runnable stream operations", "s")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return runnableDuration, batchSize, batchDuration, streamDuration, nil
}

// NewMetrics creates a new Metrics instance with registered instruments.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	runnableInvokes, runnableBatches, runnableStreams, runnableErrors, err := createCounters(meter)
	if err != nil {
		return nil, err
	}

	runnableDuration, batchSize, batchDuration, streamDuration, err := createHistograms(meter)
	if err != nil {
		return nil, err
	}

	streamChunks, err := meter.Int64Counter("runnable_stream_chunks_total",
		metric.WithDescription("Total number of chunks produced by Runnable.Stream"), metric.WithUnit("1"))
	if err != nil {
		return nil, err
	}

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/core")
	}

	return &Metrics{
		runnableInvokes:  runnableInvokes,
		runnableBatches:  runnableBatches,
		runnableStreams:  runnableStreams,
		runnableErrors:   runnableErrors,
		runnableDuration: runnableDuration,
		batchSize:        batchSize,
		batchDuration:    batchDuration,
		streamDuration:   streamDuration,
		streamChunks:     streamChunks,
		tracer:           tracer,
	}, nil
}

// RecordRunnableInvoke records metrics for a Runnable.Invoke operation.
func (m *Metrics) RecordRunnableInvoke(ctx context.Context, componentType string, duration time.Duration, err error) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("component_type", componentType),
		attribute.String("operation", "invoke"),
	}

	if m.runnableInvokes != nil {
		m.runnableInvokes.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.runnableDuration != nil {
		m.runnableDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	if err != nil && m.runnableErrors != nil {
		m.runnableErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordRunnableBatch records metrics for a Runnable.Batch operation.
func (m *Metrics) RecordRunnableBatch(
	ctx context.Context,
	componentType string,
	batchSize int,
	duration time.Duration,
	err error,
) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("component_type", componentType),
		attribute.String("operation", "batch"),
	}

	if m.runnableBatches != nil {
		m.runnableBatches.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.batchSize != nil {
		m.batchSize.Record(ctx, int64(batchSize), metric.WithAttributes(attrs...))
	}
	if m.runnableDuration != nil {
		m.runnableDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
	if m.batchDuration != nil {
		m.batchDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	if err != nil && m.runnableErrors != nil {
		m.runnableErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordRunnableStream records metrics for a Runnable.Stream operation.
func (m *Metrics) RecordRunnableStream(
	ctx context.Context,
	componentType string,
	duration time.Duration,
	chunkCount int,
	err error,
) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("component_type", componentType),
		attribute.String("operation", "stream"),
	}

	if m.runnableStreams != nil {
		m.runnableStreams.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.streamDuration != nil {
		m.streamDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
	if m.streamChunks != nil {
		m.streamChunks.Add(ctx, int64(chunkCount), metric.WithAttributes(attrs...))
	}
	if m.runnableDuration != nil {
		m.runnableDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	if err != nil && m.runnableErrors != nil {
		m.runnableErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
// It uses sync.Once to ensure thread-safe initialization.
//
// Example:
//
//	meter := otel.Meter("beluga-core")
//	tracer := otel.Tracer("beluga-core")
//	core.InitMetrics(meter, tracer)
//	metrics := core.GetMetrics()
//	if metrics != nil {
//	    metrics.RecordRunnableInvoke(ctx, "component_type", duration, err)
//	}
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if tracer == nil {
			tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/core")
		}
		metrics, err := NewMetrics(meter, tracer)
		if err != nil {
			// If metrics creation fails, use nil (callers should check)
			globalMetrics = nil
			return
		}
		globalMetrics = metrics
	})
}

// GetMetrics returns the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
// Returns nil if InitMetrics has not been called or if initialization failed.
//
// Example:
//
//	metrics := core.GetMetrics()
//	if metrics != nil {
//	    metrics.RecordRunnableInvoke(ctx, "component_type", duration, err)
//	}
func GetMetrics() *Metrics {
	return globalMetrics
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("core"),
	}
}
