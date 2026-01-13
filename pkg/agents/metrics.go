package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Package metrics provides metrics collection for the agents package.

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

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

	// Streaming metrics
	streamingLatency  metric.Float64Histogram
	streamingDuration metric.Float64Histogram
	streamingChunks   metric.Int64Counter

	// Tracer
	tracer trace.Tracer
}

// NewMetrics creates a new Metrics instance with OpenTelemetry metrics.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	m := &Metrics{tracer: tracer}

	var err error

	// Initialize agent metrics
	m.agentCreations, err = meter.Int64Counter(
		"agents_created_total",
		metric.WithDescription("Total number of agents created"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agentCreations metric: %w", err)
	}

	m.agentExecutions, err = meter.Int64Counter(
		"agent_executions_total",
		metric.WithDescription("Total number of agent executions"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agentExecutions metric: %w", err)
	}

	m.agentErrors, err = meter.Int64Counter(
		"agent_errors_total",
		metric.WithDescription("Total number of agent errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agentErrors metric: %w", err)
	}

	m.agentExecutionTime, err = meter.Float64Histogram(
		"agent_execution_duration_seconds",
		metric.WithDescription("Duration of agent executions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agentExecutionTime metric: %w", err)
	}

	// Initialize executor metrics
	m.executorRuns, err = meter.Int64Counter(
		"executor_runs_total",
		metric.WithDescription("Total number of executor runs"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executorRuns metric: %w", err)
	}

	m.executorErrors, err = meter.Int64Counter(
		"executor_errors_total",
		metric.WithDescription("Total number of executor errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executorErrors metric: %w", err)
	}

	m.executorRunTime, err = meter.Float64Histogram(
		"executor_run_duration_seconds",
		metric.WithDescription("Duration of executor runs"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executorRunTime metric: %w", err)
	}

	m.stepsExecuted, err = meter.Int64Counter(
		"steps_executed_total",
		metric.WithDescription("Total number of steps executed"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create stepsExecuted metric: %w", err)
	}

	// Initialize tool metrics
	m.toolCalls, err = meter.Int64Counter(
		"tool_calls_total",
		metric.WithDescription("Total number of tool calls"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create toolCalls metric: %w", err)
	}

	m.toolErrors, err = meter.Int64Counter(
		"tool_errors_total",
		metric.WithDescription("Total number of tool errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create toolErrors metric: %w", err)
	}

	m.toolExecutionTime, err = meter.Float64Histogram(
		"tool_execution_duration_seconds",
		metric.WithDescription("Duration of tool executions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create toolExecutionTime metric: %w", err)
	}

	// Initialize planning metrics
	m.planningCalls, err = meter.Int64Counter(
		"planning_calls_total",
		metric.WithDescription("Total number of planning calls"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create planningCalls metric: %w", err)
	}

	m.planningErrors, err = meter.Int64Counter(
		"planning_errors_total",
		metric.WithDescription("Total number of planning errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create planningErrors metric: %w", err)
	}

	m.planningTime, err = meter.Float64Histogram(
		"planning_duration_seconds",
		metric.WithDescription("Duration of planning operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create planningTime metric: %w", err)
	}

	// Initialize streaming metrics
	m.streamingLatency, err = meter.Float64Histogram(
		"agent.streaming.latency",
		metric.WithDescription("Time from input to first streaming chunk"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create streamingLatency metric: %w", err)
	}

	m.streamingDuration, err = meter.Float64Histogram(
		"agent.streaming.duration",
		metric.WithDescription("Total duration of streaming operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create streamingDuration metric: %w", err)
	}

	m.streamingChunks, err = meter.Int64Counter(
		"agent.streaming.chunks.count",
		metric.WithDescription("Total number of streaming chunks produced"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create streamingChunks metric: %w", err)
	}

	return m, nil
}

// Helper functions are no longer needed with the compatibility layer

// Agent creation metrics.
func (m *Metrics) RecordAgentCreation(ctx context.Context, agentType string) {
	if m == nil || m.agentCreations == nil {
		return
	}
	m.agentCreations.Add(ctx, 1, metric.WithAttributes(
		attribute.String("agent_type", agentType),
	))
}

// Agent execution metrics.
func (m *Metrics) RecordAgentExecution(ctx context.Context, agentName, agentType string, duration time.Duration, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("agent_name", agentName),
		attribute.String("agent_type", agentType),
		attribute.Bool("success", success),
	)

	if m.agentExecutions != nil {
		m.agentExecutions.Add(ctx, 1, attrs)
	}
	if m.agentExecutionTime != nil {
		m.agentExecutionTime.Record(ctx, duration.Seconds(), attrs)
	}
	if !success && m.agentErrors != nil {
		m.agentErrors.Add(ctx, 1, attrs)
	}
}

// Executor metrics.
func (m *Metrics) RecordExecutorRun(ctx context.Context, executorType string, duration time.Duration, steps int, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("executor_type", executorType),
		attribute.Bool("success", success),
	)

	if m.executorRuns != nil {
		m.executorRuns.Add(ctx, 1, attrs)
	}
	if m.executorRunTime != nil {
		m.executorRunTime.Record(ctx, duration.Seconds(), attrs)
	}
	if m.stepsExecuted != nil {
		m.stepsExecuted.Add(ctx, int64(steps), attrs)
	}
	if !success && m.executorErrors != nil {
		m.executorErrors.Add(ctx, 1, attrs)
	}
}

// Tool metrics.
func (m *Metrics) RecordToolCall(ctx context.Context, toolName string, duration time.Duration, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("tool_name", toolName),
		attribute.Bool("success", success),
	)

	if m.toolCalls != nil {
		m.toolCalls.Add(ctx, 1, attrs)
	}
	if m.toolExecutionTime != nil {
		m.toolExecutionTime.Record(ctx, duration.Seconds(), attrs)
	}
	if !success && m.toolErrors != nil {
		m.toolErrors.Add(ctx, 1, attrs)
	}
}

// Planning metrics.
func (m *Metrics) RecordPlanningCall(ctx context.Context, agentName string, duration time.Duration, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("agent_name", agentName),
		attribute.Bool("success", success),
	)

	if m.planningCalls != nil {
		m.planningCalls.Add(ctx, 1, attrs)
	}
	if m.planningTime != nil {
		m.planningTime.Record(ctx, duration.Seconds(), attrs)
	}
	if !success && m.planningErrors != nil {
		m.planningErrors.Add(ctx, 1, attrs)
	}
}

// Streaming metrics.
// RecordStreamingOperation records streaming operation metrics (latency and duration).
func (m *Metrics) RecordStreamingOperation(ctx context.Context, agentName string, latency, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("agent_name", agentName),
	)

	if m.streamingLatency != nil {
		m.streamingLatency.Record(ctx, latency.Seconds(), attrs)
	}
	if m.streamingDuration != nil {
		m.streamingDuration.Record(ctx, duration.Seconds(), attrs)
	}
}

// RecordStreamingChunk records that a streaming chunk was produced.
func (m *Metrics) RecordStreamingChunk(ctx context.Context, agentName string) {
	if m == nil || m.streamingChunks == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("agent_name", agentName),
	)

	m.streamingChunks.Add(ctx, 1, attrs)
}

// Tracing helpers.
// Spans returned by these methods must be ended by the caller using span.End().
func (m *Metrics) StartAgentSpan(ctx context.Context, agentName, operation string) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, "agent."+operation,
		trace.WithAttributes(
			attribute.String("agent.name", agentName),
		),
	)
	return ctx, span
}

func (m *Metrics) StartExecutorSpan(ctx context.Context, executorType, operation string) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, "executor."+operation,
		trace.WithAttributes(
			attribute.String("executor.type", executorType),
		),
	)
	return ctx, span
}

func (m *Metrics) StartToolSpan(ctx context.Context, toolName, operation string) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, "tool."+operation,
		trace.WithAttributes(
			attribute.String("tool.name", toolName),
		),
	)
	return ctx, span
}

// InitMetrics initializes the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if tracer == nil {
			tracer = otel.Tracer("beluga-agents")
		}
		metrics, err := NewMetrics(meter, tracer)
		if err != nil {
			// If metrics creation fails, use no-op metrics
			globalMetrics = NoOpMetrics()
			return
		}
		globalMetrics = metrics
	})
}

// GetMetrics returns the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func GetMetrics() *Metrics {
	return globalMetrics
}

// DefaultMetrics creates a metrics instance with default meter and tracer.
// Deprecated: Use InitMetrics(meter) and GetMetrics() instead for consistency.
func DefaultMetrics() *Metrics {
	meter := otel.Meter("beluga-agents")
	tracer := otel.Tracer("beluga-agents")
	metrics, err := NewMetrics(meter, tracer)
	if err != nil {
		// If metrics creation fails, return no-op metrics
		return NoOpMetrics()
	}
	return metrics
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
