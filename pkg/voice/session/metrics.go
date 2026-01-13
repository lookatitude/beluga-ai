package session

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
	RecordSessionStart(ctx context.Context, sessionID string, duration time.Duration)
	RecordSessionStop(ctx context.Context, sessionID string, duration time.Duration)
	RecordSessionError(ctx context.Context, sessionID, errorCode string, duration time.Duration)
	IncrementActiveSessions(ctx context.Context)
	DecrementActiveSessions(ctx context.Context)
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("voice/session"),
	}
}

// Metrics contains all the metrics for Session operations.
type Metrics struct {
	sessionsStarted  metric.Int64Counter
	sessionsStopped  metric.Int64Counter
	sessionsActive   metric.Int64UpDownCounter
	errors           metric.Int64Counter
	sessionDuration  metric.Float64Histogram
	operationLatency metric.Float64Histogram

	// Agent-specific metrics
	agentLatency           metric.Float64Histogram
	agentStreamingDuration metric.Float64Histogram
	agentToolExecutionTime metric.Float64Histogram
	tracer                 trace.Tracer
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) *Metrics {
	m := &Metrics{}

	m.sessionsStarted, _ = meter.Int64Counter("session.started.total", metric.WithDescription("Total sessions started"))
	m.sessionsStopped, _ = meter.Int64Counter("session.stopped.total", metric.WithDescription("Total sessions stopped"))
	m.sessionsActive, _ = meter.Int64UpDownCounter("session.active", metric.WithDescription("Active sessions"))
	m.errors, _ = meter.Int64Counter("session.errors.total", metric.WithDescription("Total Session errors"))
	m.sessionDuration, _ = meter.Float64Histogram("session.duration", metric.WithDescription("Session duration"), metric.WithUnit("s"))
	m.operationLatency, _ = meter.Float64Histogram("session.operation.latency", metric.WithDescription("Operation latency"), metric.WithUnit("s"))

	// Initialize agent-specific metrics
	m.agentLatency, _ = meter.Float64Histogram("voice.session.agent.latency", metric.WithDescription("Agent operation latency (input to first response)"), metric.WithUnit("s"))
	m.agentStreamingDuration, _ = meter.Float64Histogram("voice.session.agent.streaming.duration", metric.WithDescription("Agent streaming duration"), metric.WithUnit("s"))
	m.agentToolExecutionTime, _ = meter.Float64Histogram("voice.session.agent.tool.execution.time", metric.WithDescription("Agent tool execution time"), metric.WithUnit("s"))

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/voice/session")
	}
	m.tracer = tracer

	return m
}

// RecordSessionStart records a session start operation.
func (m *Metrics) RecordSessionStart(ctx context.Context, sessionID string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("session_id", sessionID),
	}
	if m.sessionsStarted != nil {
		m.sessionsStarted.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.sessionsActive != nil {
		m.sessionsActive.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.operationLatency != nil {
		m.operationLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// RecordSessionStop records a session stop operation.
func (m *Metrics) RecordSessionStop(ctx context.Context, sessionID string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("session_id", sessionID),
	}
	if m.sessionsStopped != nil {
		m.sessionsStopped.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.sessionsActive != nil {
		m.sessionsActive.Add(ctx, -1, metric.WithAttributes(attrs...))
	}
	if m.operationLatency != nil {
		m.operationLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// RecordSessionError records an error.
func (m *Metrics) RecordSessionError(ctx context.Context, sessionID, errorCode string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("session_id", sessionID),
		attribute.String("error_code", errorCode),
	}
	if m.errors != nil {
		m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.operationLatency != nil {
		m.operationLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}

// IncrementActiveSessions increments the active sessions counter.
func (m *Metrics) IncrementActiveSessions(ctx context.Context) {
	if m == nil || m.sessionsActive == nil {
		return
	}
	m.sessionsActive.Add(ctx, 1)
}

// DecrementActiveSessions decrements the active sessions counter.
func (m *Metrics) DecrementActiveSessions(ctx context.Context) {
	if m == nil || m.sessionsActive == nil {
		return
	}
	m.sessionsActive.Add(ctx, -1)
}

// Agent-specific metrics recording methods.

// RecordAgentOperation records agent operation latency (from input to first response).
func (m *Metrics) RecordAgentOperation(ctx context.Context, sessionID string, latency time.Duration) {
	if m == nil || m.agentLatency == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("session_id", sessionID),
	)
	m.agentLatency.Record(ctx, latency.Seconds(), attrs)
}

// RecordAgentStreamingChunk records agent streaming chunk metrics.
func (m *Metrics) RecordAgentStreamingChunk(ctx context.Context, sessionID string, duration time.Duration) {
	if m == nil || m.agentStreamingDuration == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("session_id", sessionID),
	)
	m.agentStreamingDuration.Record(ctx, duration.Seconds(), attrs)
}

// RecordAgentToolExecution records agent tool execution time.
func (m *Metrics) RecordAgentToolExecution(ctx context.Context, sessionID, toolName string, duration time.Duration) {
	if m == nil || m.agentToolExecutionTime == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("session_id", sessionID),
		attribute.String("tool_name", toolName),
	)
	m.agentToolExecutionTime.Record(ctx, duration.Seconds(), attrs)
}
