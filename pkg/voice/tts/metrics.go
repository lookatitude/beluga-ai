package tts

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricsRecorder defines the interface for recording metrics.
type MetricsRecorder interface {
	RecordGeneration(ctx context.Context, provider, model, voice string, duration time.Duration)
	RecordError(ctx context.Context, provider, model, voice, errorCode string, duration time.Duration)
	RecordStreaming(ctx context.Context, provider, model, voice string, duration time.Duration)
	IncrementActiveStreams(ctx context.Context, provider, model, voice string)
	DecrementActiveStreams(ctx context.Context, provider, model, voice string)
}

// NoOpMetrics provides a no-operation implementation for when metrics are disabled.
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-operation metrics recorder.
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordGeneration is a no-op implementation.
func (n *NoOpMetrics) RecordGeneration(ctx context.Context, provider, model, voice string, duration time.Duration) {
}

// RecordError is a no-op implementation.
func (n *NoOpMetrics) RecordError(ctx context.Context, provider, model, voice, errorCode string, duration time.Duration) {
}

// RecordStreaming is a no-op implementation.
func (n *NoOpMetrics) RecordStreaming(ctx context.Context, provider, model, voice string, duration time.Duration) {
}

// IncrementActiveStreams is a no-op implementation.
func (n *NoOpMetrics) IncrementActiveStreams(ctx context.Context, provider, model, voice string) {}

// DecrementActiveStreams is a no-op implementation.
func (n *NoOpMetrics) DecrementActiveStreams(ctx context.Context, provider, model, voice string) {}

// Metrics contains all the metrics for TTS operations.
type Metrics struct {
	generations       metric.Int64Counter
	successful        metric.Int64Counter
	errors            metric.Int64Counter
	failed            metric.Int64Counter
	streams           metric.Int64Counter
	generationLatency metric.Float64Histogram
	streamLatency     metric.Float64Histogram
	activeStreams     metric.Int64UpDownCounter
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter) *Metrics {
	m := &Metrics{}

	m.generations, _ = meter.Int64Counter("tts.generations.total", metric.WithDescription("Total TTS generations"))
	m.successful, _ = meter.Int64Counter("tts.generations.successful", metric.WithDescription("Successful TTS generations"))
	m.errors, _ = meter.Int64Counter("tts.errors.total", metric.WithDescription("Total TTS errors"))
	m.failed, _ = meter.Int64Counter("tts.generations.failed", metric.WithDescription("Failed TTS generations"))
	m.streams, _ = meter.Int64Counter("tts.streams.total", metric.WithDescription("Total TTS streams"))
	m.generationLatency, _ = meter.Float64Histogram("tts.generation.latency", metric.WithDescription("Generation latency"), metric.WithUnit("s"))
	m.streamLatency, _ = meter.Float64Histogram("tts.stream.latency", metric.WithDescription("Stream latency"), metric.WithUnit("s"))
	m.activeStreams, _ = meter.Int64UpDownCounter("tts.streams.active", metric.WithDescription("Active TTS streams"))

	return m
}

// RecordGeneration records a generation operation.
func (m *Metrics) RecordGeneration(ctx context.Context, provider, model, voice string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("voice", voice),
	}
	m.generations.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.successful.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.generationLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordError records an error.
func (m *Metrics) RecordError(ctx context.Context, provider, model, voice, errorCode string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("voice", voice),
		attribute.String("error_code", errorCode),
	}
	m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.failed.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.generationLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordStreaming records a streaming operation.
func (m *Metrics) RecordStreaming(ctx context.Context, provider, model, voice string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("voice", voice),
	}
	m.streams.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.streamLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// IncrementActiveStreams increments the active streams counter.
func (m *Metrics) IncrementActiveStreams(ctx context.Context, provider, model, voice string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("voice", voice),
	}
	m.activeStreams.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// DecrementActiveStreams decrements the active streams counter.
func (m *Metrics) DecrementActiveStreams(ctx context.Context, provider, model, voice string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("voice", voice),
	}
	m.activeStreams.Add(ctx, -1, metric.WithAttributes(attrs...))
}
