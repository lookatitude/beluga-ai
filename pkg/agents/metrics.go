package agents

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Package metrics provides metrics collection for the agents package.

// Metrics holds the metrics for the agents package.
type Metrics struct {
	// Agent metrics
	agentCreations     metric.Int64Counter
	agentExecutions    metric.Int64Counter
	agentErrors        metric.Int64Counter
	agentExecutionTime metric.Float64Histogram

	// Executor metrics
	executorRuns    metric.Int64Counter
	executorErrors  metric.Int64Counter
	executorRunTime metric.Float64Histogram
	stepsExecuted   metric.Int64Counter

	// Tool metrics
	toolCalls         metric.Int64Counter
	toolErrors        metric.Int64Counter
	toolExecutionTime metric.Float64Histogram

	// Planning metrics
	planningCalls  metric.Int64Counter
	planningErrors metric.Int64Counter
	planningTime   metric.Float64Histogram

	// Tracer
	tracer trace.Tracer
}

// NewMetrics creates a new Metrics instance with OpenTelemetry metrics.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) *Metrics {
	m := &Metrics{tracer: tracer}

	var err error

	// Initialize agent metrics
	m.agentCreations, err = meter.Int64Counter(
		"agents_created_total",
		metric.WithDescription("Total number of agents created"),
	)
	if err != nil {
		panic(err)
	}

	m.agentExecutions, err = meter.Int64Counter(
		"agent_executions_total",
		metric.WithDescription("Total number of agent executions"),
	)
	if err != nil {
		panic(err)
	}

	m.agentErrors, err = meter.Int64Counter(
		"agent_errors_total",
		metric.WithDescription("Total number of agent errors"),
	)
	if err != nil {
		panic(err)
	}

	m.agentExecutionTime, err = meter.Float64Histogram(
		"agent_execution_duration_seconds",
		metric.WithDescription("Duration of agent executions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	// Initialize executor metrics
	m.executorRuns, err = meter.Int64Counter(
		"executor_runs_total",
		metric.WithDescription("Total number of executor runs"),
	)
	if err != nil {
		panic(err)
	}

	m.executorErrors, err = meter.Int64Counter(
		"executor_errors_total",
		metric.WithDescription("Total number of executor errors"),
	)
	if err != nil {
		panic(err)
	}

	m.executorRunTime, err = meter.Float64Histogram(
		"executor_run_duration_seconds",
		metric.WithDescription("Duration of executor runs"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	m.stepsExecuted, err = meter.Int64Counter(
		"steps_executed_total",
		metric.WithDescription("Total number of steps executed"),
	)
	if err != nil {
		panic(err)
	}

	// Initialize tool metrics
	m.toolCalls, err = meter.Int64Counter(
		"tool_calls_total",
		metric.WithDescription("Total number of tool calls"),
	)
	if err != nil {
		panic(err)
	}

	m.toolErrors, err = meter.Int64Counter(
		"tool_errors_total",
		metric.WithDescription("Total number of tool errors"),
	)
	if err != nil {
		panic(err)
	}

	m.toolExecutionTime, err = meter.Float64Histogram(
		"tool_execution_duration_seconds",
		metric.WithDescription("Duration of tool executions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	// Initialize planning metrics
	m.planningCalls, err = meter.Int64Counter(
		"planning_calls_total",
		metric.WithDescription("Total number of planning calls"),
	)
	if err != nil {
		panic(err)
	}

	m.planningErrors, err = meter.Int64Counter(
		"planning_errors_total",
		metric.WithDescription("Total number of planning errors"),
	)
	if err != nil {
		panic(err)
	}

	m.planningTime, err = meter.Float64Histogram(
		"planning_duration_seconds",
		metric.WithDescription("Duration of planning operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	return m
}

// Helper functions are no longer needed with the compatibility layer

// Agent creation metrics.
func (m *Metrics) RecordAgentCreation(ctx context.Context, agentType string) {
	m.agentCreations.Add(ctx, 1, metric.WithAttributes(
		attribute.String("agent_type", agentType),
	))
}

// Agent execution metrics.
func (m *Metrics) RecordAgentExecution(ctx context.Context, agentName, agentType string, duration time.Duration, success bool) {
	attrs := metric.WithAttributes(
		attribute.String("agent_name", agentName),
		attribute.String("agent_type", agentType),
		attribute.Bool("success", success),
	)

	m.agentExecutions.Add(ctx, 1, attrs)
	m.agentExecutionTime.Record(ctx, duration.Seconds(), attrs)

	if !success {
		m.agentErrors.Add(ctx, 1, attrs)
	}
}

// Executor metrics.
func (m *Metrics) RecordExecutorRun(ctx context.Context, executorType string, duration time.Duration, steps int, success bool) {
	attrs := metric.WithAttributes(
		attribute.String("executor_type", executorType),
		attribute.Bool("success", success),
	)

	m.executorRuns.Add(ctx, 1, attrs)
	m.executorRunTime.Record(ctx, duration.Seconds(), attrs)
	m.stepsExecuted.Add(ctx, int64(steps), attrs)

	if !success {
		m.executorErrors.Add(ctx, 1, attrs)
	}
}

// Tool metrics.
func (m *Metrics) RecordToolCall(ctx context.Context, toolName string, duration time.Duration, success bool) {
	attrs := metric.WithAttributes(
		attribute.String("tool_name", toolName),
		attribute.Bool("success", success),
	)

	m.toolCalls.Add(ctx, 1, attrs)
	m.toolExecutionTime.Record(ctx, duration.Seconds(), attrs)

	if !success {
		m.toolErrors.Add(ctx, 1, attrs)
	}
}

// Planning metrics.
func (m *Metrics) RecordPlanningCall(ctx context.Context, agentName string, duration time.Duration, success bool) {
	attrs := metric.WithAttributes(
		attribute.String("agent_name", agentName),
		attribute.Bool("success", success),
	)

	m.planningCalls.Add(ctx, 1, attrs)
	m.planningTime.Record(ctx, duration.Seconds(), attrs)

	if !success {
		m.planningErrors.Add(ctx, 1, attrs)
	}
}

// Tracing helpers.
// Spans returned by these methods must be ended by the caller using span.End().
//
//nolint:spancheck // Spans are intentionally returned for caller to manage lifecycle
func (m *Metrics) StartAgentSpan(ctx context.Context, agentName, operation string) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, "agent."+operation,
		trace.WithAttributes(
			attribute.String("agent.name", agentName),
		),
	)
	return ctx, span
}

//nolint:spancheck // Spans are intentionally returned for caller to manage lifecycle
func (m *Metrics) StartExecutorSpan(ctx context.Context, executorType, operation string) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, "executor."+operation,
		trace.WithAttributes(
			attribute.String("executor.type", executorType),
		),
	)
	return ctx, span
}

//nolint:spancheck // Spans are intentionally returned for caller to manage lifecycle
func (m *Metrics) StartToolSpan(ctx context.Context, toolName, operation string) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, "tool."+operation,
		trace.WithAttributes(
			attribute.String("tool.name", toolName),
		),
	)
	return ctx, span
}

// DefaultMetrics creates a metrics instance with default meter and tracer.
func DefaultMetrics() *Metrics {
	meter := otel.Meter("beluga-agents")
	tracer := otel.Tracer("beluga-agents")
	return NewMetrics(meter, tracer)
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: noop.NewTracerProvider().Tracer("noop"),
	}
}

// Ensure Metrics implements the MetricsRecorder interface.
var _ iface.MetricsRecorder = (*Metrics)(nil)
