package stt

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricsRecorder defines the interface for recording metrics.
type MetricsRecorder interface {
	RecordTranscription(ctx context.Context, provider, model string, duration time.Duration)
	RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration)
	RecordStreaming(ctx context.Context, provider, model string, duration time.Duration)
	IncrementActiveStreams(ctx context.Context, provider, model string)
	DecrementActiveStreams(ctx context.Context, provider, model string)
}

// NoOpMetrics provides a no-operation implementation for when metrics are disabled.
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-operation metrics recorder.
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordTranscription is a no-op implementation.
func (n *NoOpMetrics) RecordTranscription(ctx context.Context, provider, model string, duration time.Duration) {
}

// RecordError is a no-op implementation.
func (n *NoOpMetrics) RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration) {
}

// RecordStreaming is a no-op implementation.
func (n *NoOpMetrics) RecordStreaming(ctx context.Context, provider, model string, duration time.Duration) {
}

// IncrementActiveStreams is a no-op implementation.
func (n *NoOpMetrics) IncrementActiveStreams(ctx context.Context, provider, model string) {}

// DecrementActiveStreams is a no-op implementation.
func (n *NoOpMetrics) DecrementActiveStreams(ctx context.Context, provider, model string) {}

// Metrics contains all the metrics for STT operations.
type Metrics struct {
	transcriptions       metric.Int64Counter
	successful           metric.Int64Counter
	errors               metric.Int64Counter
	failed               metric.Int64Counter
	streams              metric.Int64Counter
	transcriptionLatency metric.Float64Histogram
	streamLatency        metric.Float64Histogram
	activeStreams        metric.Int64UpDownCounter
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter) *Metrics {
	m := &Metrics{}

	m.transcriptions, _ = meter.Int64Counter("stt.transcriptions.total", metric.WithDescription("Total STT transcriptions"))
	m.successful, _ = meter.Int64Counter("stt.transcriptions.successful", metric.WithDescription("Successful STT transcriptions"))
	m.errors, _ = meter.Int64Counter("stt.errors.total", metric.WithDescription("Total STT errors"))
	m.failed, _ = meter.Int64Counter("stt.transcriptions.failed", metric.WithDescription("Failed STT transcriptions"))
	m.streams, _ = meter.Int64Counter("stt.streams.total", metric.WithDescription("Total STT streams"))
	m.transcriptionLatency, _ = meter.Float64Histogram("stt.transcription.latency", metric.WithDescription("Transcription latency"), metric.WithUnit("s"))
	m.streamLatency, _ = meter.Float64Histogram("stt.stream.latency", metric.WithDescription("Stream latency"), metric.WithUnit("s"))
	m.activeStreams, _ = meter.Int64UpDownCounter("stt.streams.active", metric.WithDescription("Active STT streams"))

	return m
}

// RecordTranscription records a transcription operation.
func (m *Metrics) RecordTranscription(ctx context.Context, provider, model string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}
	m.transcriptions.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.successful.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.transcriptionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordError records an error.
func (m *Metrics) RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("error_code", errorCode),
	}
	m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.failed.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.transcriptionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordStreaming records a streaming operation.
func (m *Metrics) RecordStreaming(ctx context.Context, provider, model string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}
	m.streams.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.streamLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// IncrementActiveStreams increments the active streams counter.
func (m *Metrics) IncrementActiveStreams(ctx context.Context, provider, model string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}
	m.activeStreams.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// DecrementActiveStreams decrements the active streams counter.
func (m *Metrics) DecrementActiveStreams(ctx context.Context, provider, model string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}
	m.activeStreams.Add(ctx, -1, metric.WithAttributes(attrs...))
}
