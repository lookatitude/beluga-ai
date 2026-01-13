package transport

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
	RecordConnection(ctx context.Context, provider string, duration time.Duration, success bool)
	RecordDisconnection(ctx context.Context, provider string, duration time.Duration)
	RecordAudioSent(ctx context.Context, provider string, bytes int64)
	RecordAudioReceived(ctx context.Context, provider string, bytes int64)
	RecordError(ctx context.Context, provider, errorCode string, duration time.Duration)
	IncrementConnections(ctx context.Context, provider string)
	DecrementConnections(ctx context.Context, provider string)
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("voice/transport"),
	}
}

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
	tracer            trace.Tracer
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) *Metrics {
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

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/voice/transport")
	}
	m.tracer = tracer

	return m
}

// RecordConnection records a connection operation.
func (m *Metrics) RecordConnection(ctx context.Context, provider string, duration time.Duration, success bool) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.Bool("success", success),
	}
	if m.connections != nil {
		m.connections.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.connectionLatency != nil {
		m.connectionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
	if success && m.activeConnections != nil {
		m.activeConnections.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordDisconnection records a disconnection operation.
func (m *Metrics) RecordDisconnection(ctx context.Context, provider string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	if m.disconnections != nil {
		m.disconnections.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.activeConnections != nil {
		m.activeConnections.Add(ctx, -1, metric.WithAttributes(attrs...))
	}
}

// RecordAudioSent records audio sent.
func (m *Metrics) RecordAudioSent(ctx context.Context, provider string, bytes int64) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	if m.audioSent != nil {
		m.audioSent.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.bytesSent != nil {
		m.bytesSent.Add(ctx, bytes, metric.WithAttributes(attrs...))
	}
}

// RecordAudioReceived records audio received.
func (m *Metrics) RecordAudioReceived(ctx context.Context, provider string, bytes int64) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	if m.audioReceived != nil {
		m.audioReceived.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.bytesReceived != nil {
		m.bytesReceived.Add(ctx, bytes, metric.WithAttributes(attrs...))
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
	if m.connectionLatency != nil {
		m.connectionLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// IncrementConnections increments the active connections counter.
func (m *Metrics) IncrementConnections(ctx context.Context, provider string) {
	if m == nil || m.activeConnections == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.activeConnections.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// DecrementConnections decrements the active connections counter.
func (m *Metrics) DecrementConnections(ctx context.Context, provider string) {
	if m == nil || m.activeConnections == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}
	m.activeConnections.Add(ctx, -1, metric.WithAttributes(attrs...))
}
