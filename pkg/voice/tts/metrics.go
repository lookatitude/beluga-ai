package tts

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
	RecordGeneration(ctx context.Context, provider, model, voice string, duration time.Duration)
	RecordError(ctx context.Context, provider, model, voice, errorCode string, duration time.Duration)
	RecordStreaming(ctx context.Context, provider, model, voice string, duration time.Duration)
	IncrementActiveStreams(ctx context.Context, provider, model, voice string)
	DecrementActiveStreams(ctx context.Context, provider, model, voice string)
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("voice/tts"),
	}
}

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
	tracer            trace.Tracer
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) *Metrics {
	m := &Metrics{}

	m.generations, _ = meter.Int64Counter("tts.generations.total", metric.WithDescription("Total TTS generations"))
	m.successful, _ = meter.Int64Counter("tts.generations.successful", metric.WithDescription("Successful TTS generations"))
	m.errors, _ = meter.Int64Counter("tts.errors.total", metric.WithDescription("Total TTS errors"))
	m.failed, _ = meter.Int64Counter("tts.generations.failed", metric.WithDescription("Failed TTS generations"))
	m.streams, _ = meter.Int64Counter("tts.streams.total", metric.WithDescription("Total TTS streams"))
	m.generationLatency, _ = meter.Float64Histogram("tts.generation.latency", metric.WithDescription("Generation latency"), metric.WithUnit("s"))
	m.streamLatency, _ = meter.Float64Histogram("tts.stream.latency", metric.WithDescription("Stream latency"), metric.WithUnit("s"))
	m.activeStreams, _ = meter.Int64UpDownCounter("tts.streams.active", metric.WithDescription("Active TTS streams"))

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/voice/tts")
	}
	m.tracer = tracer

	return m
}

// RecordGeneration records a generation operation.
func (m *Metrics) RecordGeneration(ctx context.Context, provider, model, voice string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("voice", voice),
	}
	if m.generations != nil {
		m.generations.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.successful != nil {
		m.successful.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.generationLatency != nil {
		m.generationLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// RecordError records an error.
func (m *Metrics) RecordError(ctx context.Context, provider, model, voice, errorCode string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("voice", voice),
		attribute.String("error_code", errorCode),
	}
	if m.errors != nil {
		m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.failed != nil {
		m.failed.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.generationLatency != nil {
		m.generationLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// RecordStreaming records a streaming operation.
func (m *Metrics) RecordStreaming(ctx context.Context, provider, model, voice string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("voice", voice),
	}
	if m.streams != nil {
		m.streams.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.streamLatency != nil {
		m.streamLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// IncrementActiveStreams increments the active streams counter.
func (m *Metrics) IncrementActiveStreams(ctx context.Context, provider, model, voice string) {
	if m == nil || m.activeStreams == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("voice", voice),
	}
	m.activeStreams.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// DecrementActiveStreams decrements the active streams counter.
func (m *Metrics) DecrementActiveStreams(ctx context.Context, provider, model, voice string) {
	if m == nil || m.activeStreams == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("voice", voice),
	}
	m.activeStreams.Add(ctx, -1, metric.WithAttributes(attrs...))
}
