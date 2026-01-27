package agent

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// Metrics holds the metrics for the convenience agent package.
type Metrics struct {
	// Agent metrics
	agentBuilds       metric.Int64Counter
	agentBuildErrors  metric.Int64Counter
	agentRuns         metric.Int64Counter
	agentRunErrors    metric.Int64Counter
	agentRunDuration  metric.Float64Histogram
	agentStreamChunks metric.Int64Counter

	// Tracer
	tracer trace.Tracer
}

// NewMetrics creates a new Metrics instance with OpenTelemetry metrics.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) *Metrics {
	m := &Metrics{tracer: tracer}

	var err error

	// Agent build metrics
	m.agentBuilds, err = meter.Int64Counter(
		"convenience_agent_builds_total",
		metric.WithDescription("Total number of convenience agents built"),
	)
	if err != nil {
		m.agentBuilds = nil
	}

	m.agentBuildErrors, err = meter.Int64Counter(
		"convenience_agent_build_errors_total",
		metric.WithDescription("Total number of convenience agent build errors"),
	)
	if err != nil {
		m.agentBuildErrors = nil
	}

	// Agent run metrics
	m.agentRuns, err = meter.Int64Counter(
		"convenience_agent_runs_total",
		metric.WithDescription("Total number of convenience agent runs"),
	)
	if err != nil {
		m.agentRuns = nil
	}

	m.agentRunErrors, err = meter.Int64Counter(
		"convenience_agent_run_errors_total",
		metric.WithDescription("Total number of convenience agent run errors"),
	)
	if err != nil {
		m.agentRunErrors = nil
	}

	m.agentRunDuration, err = meter.Float64Histogram(
		"convenience_agent_run_duration_seconds",
		metric.WithDescription("Duration of convenience agent runs"),
		metric.WithUnit("s"),
	)
	if err != nil {
		m.agentRunDuration = nil
	}

	// Streaming metrics
	m.agentStreamChunks, err = meter.Int64Counter(
		"convenience_agent_stream_chunks_total",
		metric.WithDescription("Total number of streaming chunks produced"),
	)
	if err != nil {
		m.agentStreamChunks = nil
	}

	return m
}

// RecordBuild records a build operation.
func (m *Metrics) RecordBuild(ctx context.Context, agentType string, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("agent_type", agentType),
		attribute.Bool("success", success),
	)

	if m.agentBuilds != nil {
		m.agentBuilds.Add(ctx, 1, attrs)
	}
	if !success && m.agentBuildErrors != nil {
		m.agentBuildErrors.Add(ctx, 1, attrs)
	}
}

// RecordRun records a run operation.
func (m *Metrics) RecordRun(ctx context.Context, agentName string, duration time.Duration, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("agent_name", agentName),
		attribute.Bool("success", success),
	)

	if m.agentRuns != nil {
		m.agentRuns.Add(ctx, 1, attrs)
	}
	if m.agentRunDuration != nil {
		m.agentRunDuration.Record(ctx, duration.Seconds(), attrs)
	}
	if !success && m.agentRunErrors != nil {
		m.agentRunErrors.Add(ctx, 1, attrs)
	}
}

// RecordStreamChunk records a streaming chunk.
func (m *Metrics) RecordStreamChunk(ctx context.Context, agentName string) {
	if m == nil || m.agentStreamChunks == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("agent_name", agentName),
	)
	m.agentStreamChunks.Add(ctx, 1, attrs)
}

// StartBuildSpan starts a tracing span for build operations.
func (m *Metrics) StartBuildSpan(ctx context.Context, agentType string) (context.Context, trace.Span) {
	if m == nil || m.tracer == nil {
		return ctx, nil
	}
	return m.tracer.Start(ctx, "convenience.agent.build",
		trace.WithAttributes(
			attribute.String("agent.type", agentType),
		),
	)
}

// StartRunSpan starts a tracing span for run operations.
func (m *Metrics) StartRunSpan(ctx context.Context, agentName, operation string) (context.Context, trace.Span) {
	if m == nil || m.tracer == nil {
		return ctx, nil
	}
	return m.tracer.Start(ctx, "convenience.agent."+operation,
		trace.WithAttributes(
			attribute.String("agent.name", agentName),
		),
	)
}

// InitMetrics initializes the global metrics instance.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if tracer == nil {
			tracer = otel.Tracer("beluga-convenience-agent")
		}
		globalMetrics = NewMetrics(meter, tracer)
	})
}

// GetMetrics returns the global metrics instance.
func GetMetrics() *Metrics {
	return globalMetrics
}

// DefaultMetrics creates a metrics instance with default meter and tracer.
func DefaultMetrics() *Metrics {
	meter := otel.Meter("beluga-convenience-agent")
	tracer := otel.Tracer("beluga-convenience-agent")
	return NewMetrics(meter, tracer)
}

// NoOpMetrics returns a metrics instance that does nothing.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: noop.NewTracerProvider().Tracer("noop"),
	}
}
