package noise

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"time"
)

// MetricsRecorder defines the interface for recording metrics
type MetricsRecorder interface {
	RecordProcessing(ctx context.Context, provider string, duration time.Duration, inputSize, outputSize int64)
	RecordError(ctx context.Context, provider, errorCode string, duration time.Duration)
	IncrementProcessedFrames(ctx context.Context, provider string)
}

// NoOpMetrics provides a no-operation implementation for when metrics are disabled
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-operation metrics recorder
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordProcessing is a no-op implementation
func (n *NoOpMetrics) RecordProcessing(ctx context.Context, provider string, duration time.Duration, inputSize, outputSize int64) {
}

// RecordError is a no-op implementation
func (n *NoOpMetrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {
}

// IncrementProcessedFrames is a no-op implementation
func (n *NoOpMetrics) IncrementProcessedFrames(ctx context.Context, provider string) {}

// Metrics contains all the metrics for Noise Cancellation operations
type Metrics struct {
	processedFrames   metric.Int64Counter
	bytesProcessed    metric.Int64Counter
	bytesOutput       metric.Int64Counter
	errors            metric.Int64Counter
	processingLatency metric.Float64Histogram
}

// NewMetrics creates a new Metrics instance
func NewMetrics(meter metric.Meter) *Metrics {
	m := &Metrics{}

	m.processedFrames, _ = meter.Int64Counter("noise.frames.processed", metric.WithDescription("Total processed audio frames"))
	m.bytesProcessed, _ = meter.Int64Counter("noise.bytes.processed", metric.WithDescription("Total bytes processed"))
	m.bytesOutput, _ = meter.Int64Counter("noise.bytes.output", metric.WithDescription("Total bytes output"))
	m.errors, _ = meter.Int64Counter("noise.errors.total", metric.WithDescription("Total Noise Cancellation errors"))
	m.processingLatency, _ = meter.Float64Histogram("noise.processing.latency", metric.WithDescription("Processing latency"), metric.WithUnit("s"))

	return m
}

// RecordProcessing records a processing operation
func (m *Metrics) RecordProcessing(ctx context.Context, provider string, duration time.Duration, inputSize, outputSize int64) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.processedFrames.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.bytesProcessed.Add(ctx, inputSize, metric.WithAttributes(attrs...))
	m.bytesOutput.Add(ctx, outputSize, metric.WithAttributes(attrs...))
	m.processingLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordError records an error
func (m *Metrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("error_code", errorCode),
	}
	m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.processingLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// IncrementProcessedFrames increments the processed frames counter
func (m *Metrics) IncrementProcessedFrames(ctx context.Context, provider string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.processedFrames.Add(ctx, 1, metric.WithAttributes(attrs...))
}
