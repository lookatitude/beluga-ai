package backend

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics contains all the metrics for voice backend operations.
type Metrics struct {
	requestsTotal        metric.Int64Counter
	errorsTotal          metric.Int64Counter
	latencySeconds       metric.Float64Histogram
	sessionsActive       metric.Int64UpDownCounter
	sessionsTotal        metric.Int64Counter
	throughputBytes      metric.Int64Counter
	concurrentOps        metric.Int64UpDownCounter
	sessionCreationTime  metric.Float64Histogram
	throughputPerSession metric.Int64Counter
	tracer               trace.Tracer
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	m := &Metrics{}

	var err error

	m.requestsTotal, err = meter.Int64Counter(
		"backend.requests.total",
		metric.WithDescription("Total voice backend requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create requestsTotal metric: %w", err)
	}

	m.errorsTotal, err = meter.Int64Counter(
		"backend.errors.total",
		metric.WithDescription("Total voice backend errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create errorsTotal metric: %w", err)
	}

	m.latencySeconds, err = meter.Float64Histogram(
		"backend.latency.seconds",
		metric.WithDescription("Voice backend latency in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create latencySeconds metric: %w", err)
	}

	m.sessionsActive, err = meter.Int64UpDownCounter(
		"backend.sessions.active",
		metric.WithDescription("Active voice backend sessions"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sessionsActive metric: %w", err)
	}

	m.sessionsTotal, err = meter.Int64Counter(
		"backend.sessions.total",
		metric.WithDescription("Total voice backend sessions created"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sessionsTotal metric: %w", err)
	}

	m.throughputBytes, err = meter.Int64Counter(
		"backend.throughput.bytes",
		metric.WithDescription("Voice backend throughput in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create throughputBytes metric: %w", err)
	}

	m.concurrentOps, err = meter.Int64UpDownCounter(
		"backend.concurrent.operations",
		metric.WithDescription("Concurrent voice backend operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create concurrentOps metric: %w", err)
	}

	m.sessionCreationTime, err = meter.Float64Histogram(
		"backend.session.creation.time.seconds",
		metric.WithDescription("Time to create a voice session in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sessionCreationTime metric: %w", err)
	}

	m.throughputPerSession, err = meter.Int64Counter(
		"backend.throughput.per.session.bytes",
		metric.WithDescription("Audio processing throughput per session in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create throughputPerSession metric: %w", err)
	}

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/voice/backend")
	}
	m.tracer = tracer

	return m, nil
}

// RecordRequest records a request metric.
func (m *Metrics) RecordRequest(ctx context.Context, provider string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider))
	if m.requestsTotal != nil {
		m.requestsTotal.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
	if m.latencySeconds != nil {
		m.latencySeconds.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
	}
}

// RecordError records an error metric.
func (m *Metrics) RecordError(ctx context.Context, provider, errorCode string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := attribute.NewSet(
		attribute.String("provider", provider),
		attribute.String("error_code", errorCode),
	)
	if m.errorsTotal != nil {
		m.errorsTotal.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
	if duration > 0 && m.latencySeconds != nil {
		m.latencySeconds.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
	}
}

// RecordLatency records a latency metric.
func (m *Metrics) RecordLatency(ctx context.Context, provider string, duration time.Duration) {
	if m == nil || m.latencySeconds == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.latencySeconds.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
}

// IncrementActiveSessions increments the active sessions counter.
func (m *Metrics) IncrementActiveSessions(ctx context.Context, provider string) {
	if m == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider))
	if m.sessionsActive != nil {
		m.sessionsActive.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
	if m.sessionsTotal != nil {
		m.sessionsTotal.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
}

// DecrementActiveSessions decrements the active sessions counter.
func (m *Metrics) DecrementActiveSessions(ctx context.Context, provider string) {
	if m == nil || m.sessionsActive == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.sessionsActive.Add(ctx, -1, metric.WithAttributeSet(attrs))
}

// RecordThroughput records throughput in bytes.
func (m *Metrics) RecordThroughput(ctx context.Context, provider string, bytes int64) {
	if m == nil || m.throughputBytes == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.throughputBytes.Add(ctx, bytes, metric.WithAttributeSet(attrs))
}

// IncrementConcurrentOps increments the concurrent operations counter.
func (m *Metrics) IncrementConcurrentOps(ctx context.Context, provider string) {
	if m == nil || m.concurrentOps == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.concurrentOps.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// DecrementConcurrentOps decrements the concurrent operations counter.
func (m *Metrics) DecrementConcurrentOps(ctx context.Context, provider string) {
	if m == nil || m.concurrentOps == nil {
		return
	}
	m.concurrentOps.Add(context.Background(), -1)
}

// RecordSessionCreationTime records the time taken to create a session.
func (m *Metrics) RecordSessionCreationTime(ctx context.Context, provider string, duration time.Duration) {
	if m == nil || m.sessionCreationTime == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider))
	m.sessionCreationTime.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
}

// RecordThroughputPerSession records throughput for a specific session.
func (m *Metrics) RecordThroughputPerSession(ctx context.Context, provider, sessionID string, bytes int64) {
	if m == nil || m.throughputPerSession == nil {
		return
	}
	attrs := attribute.NewSet(
		attribute.String("provider", provider),
		attribute.String("session_id", sessionID),
	)
	m.throughputPerSession.Add(ctx, bytes, metric.WithAttributeSet(attrs))
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("voice/backend"),
	}
}
