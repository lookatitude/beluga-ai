package transport

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricsRecorder defines the interface for recording metrics.
type MetricsRecorder interface {
	RecordConnection(ctx context.Context, provider string, duration time.Duration, success bool)
	RecordDisconnection(ctx context.Context, provider string, duration time.Duration)
	RecordAudioSent(ctx context.Context, provider string, bytes int64)
	RecordAudioReceived(ctx context.Context, provider string, bytes int64)
	RecordError(ctx context.Context, provider, errorCode string, duration time.Duration)
	IncrementConnections(ctx context.Context, provider string)
	DecrementConnections(ctx context.Context, provider string)
}

// NoOpMetrics provides a no-operation implementation for when metrics are disabled.
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-operation metrics recorder.
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordConnection is a no-op implementation.
func (n *NoOpMetrics) RecordConnection(ctx context.Context, provider string, duration time.Duration, success bool) {
}

// RecordDisconnection is a no-op implementation.
func (n *NoOpMetrics) RecordDisconnection(ctx context.Context, provider string, duration time.Duration) {
}

// RecordAudioSent is a no-op implementation.
func (n *NoOpMetrics) RecordAudioSent(ctx context.Context, provider string, bytes int64) {
}

// RecordAudioReceived is a no-op implementation.
func (n *NoOpMetrics) RecordAudioReceived(ctx context.Context, provider string, bytes int64) {
}

// RecordError is a no-op implementation.
func (n *NoOpMetrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {
}

// IncrementConnections is a no-op implementation.
func (n *NoOpMetrics) IncrementConnections(ctx context.Context, provider string) {}

// DecrementConnections is a no-op implementation.
func (n *NoOpMetrics) DecrementConnections(ctx context.Context, provider string) {}

// Metrics contains all the metrics for Transport operations.
type Metrics struct {
	connections       metric.Int64Counter
	disconnections    metric.Int64Counter
	audioSent         metric.Int64Counter
	audioReceived     metric.Int64Counter
	bytesSent         metric.Int64Counter
	bytesReceived     metric.Int64Counter
	errors            metric.Int64Counter
	connectionLatency metric.Float64Histogram
	activeConnections metric.Int64UpDownCounter
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter) *Metrics {
	m := &Metrics{}

	m.connections, _ = meter.Int64Counter("transport.connections.total", metric.WithDescription("Total connections"))
	m.disconnections, _ = meter.Int64Counter("transport.disconnections.total", metric.WithDescription("Total disconnections"))
	m.audioSent, _ = meter.Int64Counter("transport.audio.sent", metric.WithDescription("Audio packets sent"))
	m.audioReceived, _ = meter.Int64Counter("transport.audio.received", metric.WithDescription("Audio packets received"))
	m.bytesSent, _ = meter.Int64Counter("transport.bytes.sent", metric.WithDescription("Bytes sent"))
	m.bytesReceived, _ = meter.Int64Counter("transport.bytes.received", metric.WithDescription("Bytes received"))
	m.errors, _ = meter.Int64Counter("transport.errors.total", metric.WithDescription("Total Transport errors"))
	m.connectionLatency, _ = meter.Float64Histogram("transport.connection.latency", metric.WithDescription("Connection latency"), metric.WithUnit("s"))
	m.activeConnections, _ = meter.Int64UpDownCounter("transport.connections.active", metric.WithDescription("Active connections"))

	return m
}

// RecordConnection records a connection operation.
func (m *Metrics) RecordConnection(ctx context.Context, provider string, duration time.Duration, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.Bool("success", success),
	}
	m.connections.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.connectionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	if success {
		m.activeConnections.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordDisconnection records a disconnection operation.
func (m *Metrics) RecordDisconnection(ctx context.Context, provider string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.disconnections.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.activeConnections.Add(ctx, -1, metric.WithAttributes(attrs...))
}

// RecordAudioSent records audio sent.
func (m *Metrics) RecordAudioSent(ctx context.Context, provider string, bytes int64) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.audioSent.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.bytesSent.Add(ctx, bytes, metric.WithAttributes(attrs...))
}

// RecordAudioReceived records audio received.
func (m *Metrics) RecordAudioReceived(ctx context.Context, provider string, bytes int64) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.audioReceived.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.bytesReceived.Add(ctx, bytes, metric.WithAttributes(attrs...))
}

// RecordError records an error.
func (m *Metrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("error_code", errorCode),
	}
	m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.connectionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// IncrementConnections increments the active connections counter.
func (m *Metrics) IncrementConnections(ctx context.Context, provider string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.activeConnections.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// DecrementConnections decrements the active connections counter.
func (m *Metrics) DecrementConnections(ctx context.Context, provider string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.activeConnections.Add(ctx, -1, metric.WithAttributes(attrs...))
}
