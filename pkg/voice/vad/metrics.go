package vad

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
	RecordProcessing(ctx context.Context, provider string, duration time.Duration, detected bool)
	RecordError(ctx context.Context, provider, errorCode string, duration time.Duration)
	IncrementProcessedFrames(ctx context.Context, provider string)
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("voice/vad"),
	}
}

// Metrics contains all the metrics for VAD operations.
type Metrics struct {
	processedFrames   metric.Int64Counter
	speechDetected    metric.Int64Counter
	silenceDetected   metric.Int64Counter
	errors            metric.Int64Counter
	processingLatency metric.Float64Histogram
	tracer            trace.Tracer
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) *Metrics {
	m := &Metrics{}

	m.processedFrames, _ = meter.Int64Counter("vad.frames.processed", metric.WithDescription("Total processed audio frames"))
	m.speechDetected, _ = meter.Int64Counter("vad.speech.detected", metric.WithDescription("Speech detected events"))
	m.silenceDetected, _ = meter.Int64Counter("vad.silence.detected", metric.WithDescription("Silence detected events"))
	m.errors, _ = meter.Int64Counter("vad.errors.total", metric.WithDescription("Total VAD errors"))
	m.processingLatency, _ = meter.Float64Histogram("vad.processing.latency", metric.WithDescription("Processing latency"), metric.WithUnit("s"))

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/voice/vad")
	}
	m.tracer = tracer

	return m
}

// RecordProcessing records a processing operation.
func (m *Metrics) RecordProcessing(ctx context.Context, provider string, duration time.Duration, detected bool) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	if m.processedFrames != nil {
		m.processedFrames.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.processingLatency != nil {
		m.processingLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	if detected {
		if m.speechDetected != nil {
			m.speechDetected.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	} else {
		if m.silenceDetected != nil {
			m.silenceDetected.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	}
}

// RecordError records an error.
func (m *Metrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("error_code", errorCode),
	}
	if m.errors != nil {
		m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.processingLatency != nil {
		m.processingLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// IncrementProcessedFrames increments the processed frames counter.
func (m *Metrics) IncrementProcessedFrames(ctx context.Context, provider string) {
	if m == nil || m.processedFrames == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.processedFrames.Add(ctx, 1, metric.WithAttributes(attrs...))
}
