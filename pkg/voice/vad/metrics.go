package vad

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricsRecorder defines the interface for recording metrics.
type MetricsRecorder interface {
	RecordProcessing(ctx context.Context, provider string, duration time.Duration, detected bool)
	RecordError(ctx context.Context, provider, errorCode string, duration time.Duration)
	IncrementProcessedFrames(ctx context.Context, provider string)
}

// NoOpMetrics provides a no-operation implementation for when metrics are disabled.
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-operation metrics recorder.
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordProcessing is a no-op implementation.
func (n *NoOpMetrics) RecordProcessing(ctx context.Context, provider string, duration time.Duration, detected bool) {
}

// RecordError is a no-op implementation.
func (n *NoOpMetrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {
}

// IncrementProcessedFrames is a no-op implementation.
func (n *NoOpMetrics) IncrementProcessedFrames(ctx context.Context, provider string) {}

// Metrics contains all the metrics for VAD operations.
type Metrics struct {
	processedFrames   metric.Int64Counter
	speechDetected    metric.Int64Counter
	silenceDetected   metric.Int64Counter
	errors            metric.Int64Counter
	processingLatency metric.Float64Histogram
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter) *Metrics {
	m := &Metrics{}

	m.processedFrames, _ = meter.Int64Counter("vad.frames.processed", metric.WithDescription("Total processed audio frames"))
	m.speechDetected, _ = meter.Int64Counter("vad.speech.detected", metric.WithDescription("Speech detected events"))
	m.silenceDetected, _ = meter.Int64Counter("vad.silence.detected", metric.WithDescription("Silence detected events"))
	m.errors, _ = meter.Int64Counter("vad.errors.total", metric.WithDescription("Total VAD errors"))
	m.processingLatency, _ = meter.Float64Histogram("vad.processing.latency", metric.WithDescription("Processing latency"), metric.WithUnit("s"))

	return m
}

// RecordProcessing records a processing operation.
func (m *Metrics) RecordProcessing(ctx context.Context, provider string, duration time.Duration, detected bool) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.processedFrames.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.processingLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if detected {
		m.speechDetected.Add(ctx, 1, metric.WithAttributes(attrs...))
	} else {
		m.silenceDetected.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordError records an error.
func (m *Metrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("error_code", errorCode),
	}
	m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.processingLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// IncrementProcessedFrames increments the processed frames counter.
func (m *Metrics) IncrementProcessedFrames(ctx context.Context, provider string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.processedFrames.Add(ctx, 1, metric.WithAttributes(attrs...))
}
