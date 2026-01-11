package backend

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics contains all the metrics for voice backend operations.
type Metrics struct {
	requestsTotal      metric.Int64Counter
	errorsTotal        metric.Int64Counter
	latencySeconds     metric.Float64Histogram
	sessionsActive        metric.Int64UpDownCounter
	sessionsTotal         metric.Int64Counter
	throughputBytes       metric.Int64Counter
	concurrentOps         metric.Int64UpDownCounter
	sessionCreationTime   metric.Float64Histogram
	throughputPerSession  metric.Int64Counter
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter) *Metrics {
	m := &Metrics{}

	m.requestsTotal, _ = meter.Int64Counter(
		"backend.requests.total",
		metric.WithDescription("Total voice backend requests"),
	) //nolint:errcheck // Metrics initialization errors are handled by returning nil or no-op metrics
	m.errorsTotal, _ = meter.Int64Counter(
		"backend.errors.total",
		metric.WithDescription("Total voice backend errors"),
	) //nolint:errcheck
	m.latencySeconds, _ = meter.Float64Histogram(
		"backend.latency.seconds",
		metric.WithDescription("Voice backend latency in seconds"),
		metric.WithUnit("s"),
	) //nolint:errcheck
	m.sessionsActive, _ = meter.Int64UpDownCounter(
		"backend.sessions.active",
		metric.WithDescription("Active voice backend sessions"),
	) //nolint:errcheck
	m.sessionsTotal, _ = meter.Int64Counter(
		"backend.sessions.total",
		metric.WithDescription("Total voice backend sessions created"),
	) //nolint:errcheck
	m.throughputBytes, _ = meter.Int64Counter(
		"backend.throughput.bytes",
		metric.WithDescription("Voice backend throughput in bytes"),
		metric.WithUnit("By"),
	) //nolint:errcheck
	m.concurrentOps, _ = meter.Int64UpDownCounter(
		"backend.concurrent.operations",
		metric.WithDescription("Concurrent voice backend operations"),
	) //nolint:errcheck
	m.sessionCreationTime, _ = meter.Float64Histogram(
		"backend.session.creation.time.seconds",
		metric.WithDescription("Time to create a voice session in seconds"),
		metric.WithUnit("s"),
	) //nolint:errcheck
	m.throughputPerSession, _ = meter.Int64Counter(
		"backend.throughput.per.session.bytes",
		metric.WithDescription("Audio processing throughput per session in bytes"),
		metric.WithUnit("By"),
	) //nolint:errcheck

	return m
}

// RecordRequest records a request metric.
func (m *Metrics) RecordRequest(ctx context.Context, provider string, duration time.Duration) {
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.requestsTotal.Add(ctx, 1, metric.WithAttributeSet(attrs))
	m.latencySeconds.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
}

// RecordError records an error metric.
func (m *Metrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {
	attrs := attribute.NewSet(
		attribute.String("provider", provider),
		attribute.String("error_code", errorCode),
	)
	m.errorsTotal.Add(ctx, 1, metric.WithAttributeSet(attrs))
	if duration > 0 {
		m.latencySeconds.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
	}
}

// RecordLatency records a latency metric.
func (m *Metrics) RecordLatency(ctx context.Context, provider string, duration time.Duration) {
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.latencySeconds.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
}

// IncrementActiveSessions increments the active sessions counter.
func (m *Metrics) IncrementActiveSessions(ctx context.Context, provider string) {
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.sessionsActive.Add(ctx, 1, metric.WithAttributeSet(attrs))
	m.sessionsTotal.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// DecrementActiveSessions decrements the active sessions counter.
func (m *Metrics) DecrementActiveSessions(ctx context.Context, provider string) {
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.sessionsActive.Add(ctx, -1, metric.WithAttributeSet(attrs))
}

// RecordThroughput records throughput in bytes.
func (m *Metrics) RecordThroughput(ctx context.Context, provider string, bytes int64) {
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.throughputBytes.Add(ctx, bytes, metric.WithAttributeSet(attrs))
}

// IncrementConcurrentOps increments the concurrent operations counter.
func (m *Metrics) IncrementConcurrentOps(ctx context.Context, provider string) {
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.concurrentOps.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// DecrementConcurrentOps decrements the concurrent operations counter.
func (m *Metrics) DecrementConcurrentOps(ctx context.Context, provider string) {
	m.concurrentOps.Add(context.Background(), -1)
}

// RecordSessionCreationTime records the time taken to create a session.
func (m *Metrics) RecordSessionCreationTime(ctx context.Context, provider string, duration time.Duration) {
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.sessionCreationTime.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
}

// RecordThroughputPerSession records throughput for a specific session.
func (m *Metrics) RecordThroughputPerSession(ctx context.Context, provider, sessionID string, bytes int64) {
	attrs := attribute.NewSet(
		attribute.String("provider", provider),
		attribute.String("session_id", sessionID),
	)
	m.throughputPerSession.Add(ctx, bytes, metric.WithAttributeSet(attrs))
}

// NoOpMetrics provides a no-operation implementation for when metrics are disabled.
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-operation metrics recorder.
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordRequest is a no-op implementation.
func (n *NoOpMetrics) RecordRequest(ctx context.Context, provider string, duration time.Duration) {}

// RecordError is a no-op implementation.
func (n *NoOpMetrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {}

// RecordLatency is a no-op implementation.
func (n *NoOpMetrics) RecordLatency(ctx context.Context, provider string, duration time.Duration) {}

// IncrementActiveSessions is a no-op implementation.
func (n *NoOpMetrics) IncrementActiveSessions(ctx context.Context, provider string) {}

// DecrementActiveSessions is a no-op implementation.
func (n *NoOpMetrics) DecrementActiveSessions(ctx context.Context, provider string) {}

// RecordThroughput is a no-op implementation.
func (n *NoOpMetrics) RecordThroughput(ctx context.Context, provider string, bytes int64) {}

// IncrementConcurrentOps is a no-op implementation.
func (n *NoOpMetrics) IncrementConcurrentOps(ctx context.Context, provider string) {}

// DecrementConcurrentOps is a no-op implementation.
func (n *NoOpMetrics) DecrementConcurrentOps(ctx context.Context, provider string) {}

// RecordSessionCreationTime is a no-op implementation.
func (n *NoOpMetrics) RecordSessionCreationTime(ctx context.Context, provider string, duration time.Duration) {}

// RecordThroughputPerSession is a no-op implementation.
func (n *NoOpMetrics) RecordThroughputPerSession(ctx context.Context, provider, sessionID string, bytes int64) {}
