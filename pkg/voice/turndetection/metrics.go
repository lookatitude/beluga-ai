package turndetection

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"time"
)

// MetricsRecorder defines the interface for recording metrics
type MetricsRecorder interface {
	RecordDetection(ctx context.Context, provider string, duration time.Duration, turnDetected bool)
	RecordError(ctx context.Context, provider, errorCode string, duration time.Duration)
	IncrementDetections(ctx context.Context, provider string)
}

// NoOpMetrics provides a no-operation implementation for when metrics are disabled
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-operation metrics recorder
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordDetection is a no-op implementation
func (n *NoOpMetrics) RecordDetection(ctx context.Context, provider string, duration time.Duration, turnDetected bool) {
}

// RecordError is a no-op implementation
func (n *NoOpMetrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {
}

// IncrementDetections is a no-op implementation
func (n *NoOpMetrics) IncrementDetections(ctx context.Context, provider string) {}

// Metrics contains all the metrics for Turn Detection operations
type Metrics struct {
	detections       metric.Int64Counter
	turnsDetected    metric.Int64Counter
	turnsNotDetected metric.Int64Counter
	errors           metric.Int64Counter
	detectionLatency metric.Float64Histogram
}

// NewMetrics creates a new Metrics instance
func NewMetrics(meter metric.Meter) *Metrics {
	m := &Metrics{}

	m.detections, _ = meter.Int64Counter("turndetection.detections.total", metric.WithDescription("Total turn detections"))
	m.turnsDetected, _ = meter.Int64Counter("turndetection.turns.detected", metric.WithDescription("Turns detected"))
	m.turnsNotDetected, _ = meter.Int64Counter("turndetection.turns.not_detected", metric.WithDescription("Turns not detected"))
	m.errors, _ = meter.Int64Counter("turndetection.errors.total", metric.WithDescription("Total Turn Detection errors"))
	m.detectionLatency, _ = meter.Float64Histogram("turndetection.detection.latency", metric.WithDescription("Detection latency"), metric.WithUnit("s"))

	return m
}

// RecordDetection records a detection operation
func (m *Metrics) RecordDetection(ctx context.Context, provider string, duration time.Duration, turnDetected bool) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.detections.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.detectionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if turnDetected {
		m.turnsDetected.Add(ctx, 1, metric.WithAttributes(attrs...))
	} else {
		m.turnsNotDetected.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordError records an error
func (m *Metrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("error_code", errorCode),
	}
	m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.detectionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// IncrementDetections increments the detections counter
func (m *Metrics) IncrementDetections(ctx context.Context, provider string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.detections.Add(ctx, 1, metric.WithAttributes(attrs...))
}
