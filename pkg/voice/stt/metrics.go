package stt

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// MetricsRecorder defines the interface for recording metrics.
type MetricsRecorder interface {
	RecordTranscription(ctx context.Context, provider, model string, duration time.Duration)
	RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration)
	RecordStreaming(ctx context.Context, provider, model string, duration time.Duration)
	IncrementActiveStreams(ctx context.Context, provider, model string)
	DecrementActiveStreams(ctx context.Context, provider, model string)
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("voice/stt"),
	}
}

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
	tracer               trace.Tracer
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) *Metrics {
	m := &Metrics{}

	m.transcriptions, _ = meter.Int64Counter("stt.transcriptions.total", metric.WithDescription("Total STT transcriptions"))
	m.successful, _ = meter.Int64Counter("stt.transcriptions.successful", metric.WithDescription("Successful STT transcriptions"))
	m.errors, _ = meter.Int64Counter("stt.errors.total", metric.WithDescription("Total STT errors"))
	m.failed, _ = meter.Int64Counter("stt.transcriptions.failed", metric.WithDescription("Failed STT transcriptions"))
	m.streams, _ = meter.Int64Counter("stt.streams.total", metric.WithDescription("Total STT streams"))
	m.transcriptionLatency, _ = meter.Float64Histogram("stt.transcription.latency", metric.WithDescription("Transcription latency"), metric.WithUnit("s"))
	m.streamLatency, _ = meter.Float64Histogram("stt.stream.latency", metric.WithDescription("Stream latency"), metric.WithUnit("s"))
	m.activeStreams, _ = meter.Int64UpDownCounter("stt.streams.active", metric.WithDescription("Active STT streams"))

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/voice/stt")
	}
	m.tracer = tracer

	return m
}

// RecordTranscription records a transcription operation.
func (m *Metrics) RecordTranscription(ctx context.Context, provider, model string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}
	if m.transcriptions != nil {
		m.transcriptions.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.successful != nil {
		m.successful.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.transcriptionLatency != nil {
		m.transcriptionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// RecordError records an error.
func (m *Metrics) RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("error_code", errorCode),
	}
	if m.errors != nil {
		m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.failed != nil {
		m.failed.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.transcriptionLatency != nil {
		m.transcriptionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// RecordStreaming records a streaming operation.
func (m *Metrics) RecordStreaming(ctx context.Context, provider, model string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}
	if m.streams != nil {
		m.streams.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.streamLatency != nil {
		m.streamLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// IncrementActiveStreams increments the active streams counter.
func (m *Metrics) IncrementActiveStreams(ctx context.Context, provider, model string) {
	if m == nil || m.activeStreams == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}
	m.activeStreams.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// DecrementActiveStreams decrements the active streams counter.
func (m *Metrics) DecrementActiveStreams(ctx context.Context, provider, model string) {
	if m == nil || m.activeStreams == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}
	m.activeStreams.Add(ctx, -1, metric.WithAttributes(attrs...))
}
