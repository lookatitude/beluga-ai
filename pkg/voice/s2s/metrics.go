package s2s

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricsRecorder defines the interface for recording metrics.
type MetricsRecorder interface {
	RecordProcess(ctx context.Context, provider, model string, duration time.Duration)
	RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration)
	RecordStreaming(ctx context.Context, provider, model string, duration time.Duration)
	IncrementActiveStreams(ctx context.Context, provider, model string)
	DecrementActiveStreams(ctx context.Context, provider, model string)
	RecordProviderUsage(ctx context.Context, provider string)
	RecordFallback(ctx context.Context, fromProvider, toProvider string)
	RecordConcurrentSessions(ctx context.Context, provider string, count int64)
	RecordReasoningMode(ctx context.Context, provider, reasoningMode string)
	RecordLatencyTarget(ctx context.Context, provider, latencyTarget string, actualLatency time.Duration)
	RecordAudioQuality(ctx context.Context, provider string, quality float64)
}

// NoOpMetrics provides a no-operation implementation for when metrics are disabled.
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-operation metrics recorder.
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordProcess is a no-op implementation.
func (n *NoOpMetrics) RecordProcess(ctx context.Context, provider, model string, duration time.Duration) {
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

// RecordProviderUsage is a no-op implementation.
func (n *NoOpMetrics) RecordProviderUsage(ctx context.Context, provider string) {}

// RecordFallback is a no-op implementation.
func (n *NoOpMetrics) RecordFallback(ctx context.Context, fromProvider, toProvider string) {}

// RecordConcurrentSessions is a no-op implementation.
func (n *NoOpMetrics) RecordConcurrentSessions(ctx context.Context, provider string, count int64) {}

// RecordReasoningMode is a no-op implementation.
func (n *NoOpMetrics) RecordReasoningMode(ctx context.Context, provider, reasoningMode string) {}

// RecordLatencyTarget is a no-op implementation.
func (n *NoOpMetrics) RecordLatencyTarget(ctx context.Context, provider, latencyTarget string, actualLatency time.Duration) {
}

// RecordAudioQuality is a no-op implementation.
func (n *NoOpMetrics) RecordAudioQuality(ctx context.Context, provider string, quality float64) {}

// Metrics contains all the metrics for S2S operations.
type Metrics struct {
	processes          metric.Int64Counter
	successful         metric.Int64Counter
	errors             metric.Int64Counter
	failed             metric.Int64Counter
	streams            metric.Int64Counter
	processLatency     metric.Float64Histogram
	streamLatency      metric.Float64Histogram
	activeStreams      metric.Int64UpDownCounter
	providerUsage      metric.Int64Counter
	fallbackEvents     metric.Int64Counter
	concurrentSessions metric.Int64Gauge
	reasoningMode      metric.Int64Counter
	latencyTarget      metric.Float64Histogram
	audioQuality       metric.Float64Histogram
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter) *Metrics {
	m := &Metrics{}

	m.processes, _ = meter.Int64Counter("s2s.processes.total", metric.WithDescription("Total S2S processes"))
	m.successful, _ = meter.Int64Counter("s2s.processes.successful", metric.WithDescription("Successful S2S processes"))
	m.errors, _ = meter.Int64Counter("s2s.errors.total", metric.WithDescription("Total S2S errors"))
	m.failed, _ = meter.Int64Counter("s2s.processes.failed", metric.WithDescription("Failed S2S processes"))
	m.streams, _ = meter.Int64Counter("s2s.streams.total", metric.WithDescription("Total S2S streams"))
	m.processLatency, _ = meter.Float64Histogram("s2s.process.latency", metric.WithDescription("Process latency"), metric.WithUnit("s"))
	m.streamLatency, _ = meter.Float64Histogram("s2s.stream.latency", metric.WithDescription("Stream latency"), metric.WithUnit("s"))
	m.activeStreams, _ = meter.Int64UpDownCounter("s2s.streams.active", metric.WithDescription("Active S2S streams"))
	m.providerUsage, _ = meter.Int64Counter("s2s.provider.usage", metric.WithDescription("Provider usage count"))
	m.fallbackEvents, _ = meter.Int64Counter("s2s.fallback.events", metric.WithDescription("Fallback events"))
	m.concurrentSessions, _ = meter.Int64Gauge("s2s.concurrent.sessions", metric.WithDescription("Concurrent sessions"))
	m.reasoningMode, _ = meter.Int64Counter("s2s.reasoning.mode", metric.WithDescription("Reasoning mode usage"))
	m.latencyTarget, _ = meter.Float64Histogram("s2s.latency.target", metric.WithDescription("Latency target vs actual"), metric.WithUnit("s"))
	m.audioQuality, _ = meter.Float64Histogram("s2s.audio.quality", metric.WithDescription("Audio quality score (0.0-1.0)"))

	return m
}

// RecordProcess records a process operation.
func (m *Metrics) RecordProcess(ctx context.Context, provider, model string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}
	m.processes.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.successful.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.processLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
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
	m.processLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
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

// RecordProviderUsage records provider usage.
func (m *Metrics) RecordProviderUsage(ctx context.Context, provider string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.providerUsage.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordFallback records a fallback event.
func (m *Metrics) RecordFallback(ctx context.Context, fromProvider, toProvider string) {
	attrs := []attribute.KeyValue{
		attribute.String("from_provider", fromProvider),
		attribute.String("to_provider", toProvider),
	}
	m.fallbackEvents.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordConcurrentSessions records concurrent session count.
func (m *Metrics) RecordConcurrentSessions(ctx context.Context, provider string, count int64) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.concurrentSessions.Record(ctx, count, metric.WithAttributes(attrs...))
}

// RecordReasoningMode records reasoning mode usage.
func (m *Metrics) RecordReasoningMode(ctx context.Context, provider, reasoningMode string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("reasoning_mode", reasoningMode),
	}
	m.reasoningMode.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordLatencyTarget records latency target vs actual latency.
func (m *Metrics) RecordLatencyTarget(ctx context.Context, provider, latencyTarget string, actualLatency time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("latency_target", latencyTarget),
	}
	m.latencyTarget.Record(ctx, actualLatency.Seconds(), metric.WithAttributes(attrs...))
}

// RecordAudioQuality records audio quality score.
func (m *Metrics) RecordAudioQuality(ctx context.Context, provider string, quality float64) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.audioQuality.Record(ctx, quality, metric.WithAttributes(attrs...))
}
