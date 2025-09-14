// Package core provides metrics definitions for the Beluga AI framework.
// It defines standard metrics for Runnable executions and other core operations.
package core

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
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
}

// NewMetrics creates a new Metrics instance with registered instruments.
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	runnableInvokes, err := meter.Int64Counter(
		"runnable_invokes_total",
		metric.WithDescription("Total number of Runnable.Invoke calls"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	runnableBatches, err := meter.Int64Counter(
		"runnable_batches_total",
		metric.WithDescription("Total number of Runnable.Batch calls"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	runnableStreams, err := meter.Int64Counter(
		"runnable_streams_total",
		metric.WithDescription("Total number of Runnable.Stream calls"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	runnableErrors, err := meter.Int64Counter(
		"runnable_errors_total",
		metric.WithDescription("Total number of Runnable execution errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	runnableDuration, err := meter.Float64Histogram(
		"runnable_duration_seconds",
		metric.WithDescription("Duration of Runnable operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	batchSize, err := meter.Int64Histogram(
		"runnable_batch_size",
		metric.WithDescription("Size of Runnable batch operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	batchDuration, err := meter.Float64Histogram(
		"runnable_batch_duration_seconds",
		metric.WithDescription("Duration of Runnable batch operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	streamDuration, err := meter.Float64Histogram(
		"runnable_stream_duration_seconds",
		metric.WithDescription("Duration of Runnable stream operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	streamChunks, err := meter.Int64Counter(
		"runnable_stream_chunks_total",
		metric.WithDescription("Total number of chunks produced by Runnable.Stream"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
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
func (m *Metrics) RecordRunnableBatch(ctx context.Context, componentType string, batchSize int, duration time.Duration, err error) {
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
func (m *Metrics) RecordRunnableStream(ctx context.Context, componentType string, duration time.Duration, chunkCount int, err error) {
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

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{}
}
