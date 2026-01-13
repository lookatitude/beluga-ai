package turndetection

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
	RecordDetection(ctx context.Context, provider string, duration time.Duration, turnDetected bool)
	RecordError(ctx context.Context, provider, errorCode string, duration time.Duration)
	IncrementDetections(ctx context.Context, provider string)
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("voice/turndetection"),
	}
}

// Metrics contains all the metrics for Turn Detection operations.
type Metrics struct {
	detections       metric.Int64Counter
	turnsDetected    metric.Int64Counter
	turnsNotDetected metric.Int64Counter
	errors           metric.Int64Counter
	detectionLatency metric.Float64Histogram
	tracer           trace.Tracer
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) *Metrics {
	m := &Metrics{}

	m.detections, _ = meter.Int64Counter("turndetection.detections.total", metric.WithDescription("Total turn detections"))
	m.turnsDetected, _ = meter.Int64Counter("turndetection.turns.detected", metric.WithDescription("Turns detected"))
	m.turnsNotDetected, _ = meter.Int64Counter("turndetection.turns.not_detected", metric.WithDescription("Turns not detected"))
	m.errors, _ = meter.Int64Counter("turndetection.errors.total", metric.WithDescription("Total Turn Detection errors"))
	m.detectionLatency, _ = meter.Float64Histogram("turndetection.detection.latency", metric.WithDescription("Detection latency"), metric.WithUnit("s"))

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/voice/turndetection")
	}
	m.tracer = tracer

	return m
}

// RecordDetection records a detection operation.
func (m *Metrics) RecordDetection(ctx context.Context, provider string, duration time.Duration, turnDetected bool) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	if m.detections != nil {
		m.detections.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.detectionLatency != nil {
		m.detectionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	if turnDetected {
		if m.turnsDetected != nil {
			m.turnsDetected.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	} else {
		if m.turnsNotDetected != nil {
			m.turnsNotDetected.Add(ctx, 1, metric.WithAttributes(attrs...))
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
	if m.detectionLatency != nil {
		m.detectionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// IncrementDetections increments the detections counter.
func (m *Metrics) IncrementDetections(ctx context.Context, provider string) {
	if m == nil || m.detections == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.detections.Add(ctx, 1, metric.WithAttributes(attrs...))
}
