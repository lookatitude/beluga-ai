package session

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"time"
)

// MetricsRecorder defines the interface for recording metrics
type MetricsRecorder interface {
	RecordSessionStart(ctx context.Context, sessionID string, duration time.Duration)
	RecordSessionStop(ctx context.Context, sessionID string, duration time.Duration)
	RecordSessionError(ctx context.Context, sessionID, errorCode string, duration time.Duration)
	IncrementActiveSessions(ctx context.Context)
	DecrementActiveSessions(ctx context.Context)
}

// NoOpMetrics provides a no-operation implementation for when metrics are disabled
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-operation metrics recorder
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordSessionStart is a no-op implementation
func (n *NoOpMetrics) RecordSessionStart(ctx context.Context, sessionID string, duration time.Duration) {
}

// RecordSessionStop is a no-op implementation
func (n *NoOpMetrics) RecordSessionStop(ctx context.Context, sessionID string, duration time.Duration) {
}

// RecordSessionError is a no-op implementation
func (n *NoOpMetrics) RecordSessionError(ctx context.Context, sessionID, errorCode string, duration time.Duration) {
}

// IncrementActiveSessions is a no-op implementation
func (n *NoOpMetrics) IncrementActiveSessions(ctx context.Context) {}

// DecrementActiveSessions is a no-op implementation
func (n *NoOpMetrics) DecrementActiveSessions(ctx context.Context) {}

// Metrics contains all the metrics for Session operations
type Metrics struct {
	sessionsStarted  metric.Int64Counter
	sessionsStopped  metric.Int64Counter
	sessionsActive   metric.Int64UpDownCounter
	errors           metric.Int64Counter
	sessionDuration  metric.Float64Histogram
	operationLatency metric.Float64Histogram
}

// NewMetrics creates a new Metrics instance
func NewMetrics(meter metric.Meter) *Metrics {
	m := &Metrics{}

	m.sessionsStarted, _ = meter.Int64Counter("session.started.total", metric.WithDescription("Total sessions started"))
	m.sessionsStopped, _ = meter.Int64Counter("session.stopped.total", metric.WithDescription("Total sessions stopped"))
	m.sessionsActive, _ = meter.Int64UpDownCounter("session.active", metric.WithDescription("Active sessions"))
	m.errors, _ = meter.Int64Counter("session.errors.total", metric.WithDescription("Total Session errors"))
	m.sessionDuration, _ = meter.Float64Histogram("session.duration", metric.WithDescription("Session duration"), metric.WithUnit("s"))
	m.operationLatency, _ = meter.Float64Histogram("session.operation.latency", metric.WithDescription("Operation latency"), metric.WithUnit("s"))

	return m
}

// RecordSessionStart records a session start operation
func (m *Metrics) RecordSessionStart(ctx context.Context, sessionID string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("session_id", sessionID),
	}
	m.sessionsStarted.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.sessionsActive.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.operationLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordSessionStop records a session stop operation
func (m *Metrics) RecordSessionStop(ctx context.Context, sessionID string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("session_id", sessionID),
	}
	m.sessionsStopped.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.sessionsActive.Add(ctx, -1, metric.WithAttributes(attrs...))
	m.operationLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordSessionError records an error
func (m *Metrics) RecordSessionError(ctx context.Context, sessionID, errorCode string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("session_id", sessionID),
		attribute.String("error_code", errorCode),
	}
	m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.operationLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// IncrementActiveSessions increments the active sessions counter
func (m *Metrics) IncrementActiveSessions(ctx context.Context) {
	m.sessionsActive.Add(ctx, 1)
}

// DecrementActiveSessions decrements the active sessions counter
func (m *Metrics) DecrementActiveSessions(ctx context.Context) {
	m.sessionsActive.Add(ctx, -1)
}
