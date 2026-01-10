package orchestration

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds all orchestration-related metrics.
type Metrics struct {
	// Chain metrics
	chainExecutions metric.Int64Counter
	chainDuration   metric.Float64Histogram
	chainErrors     metric.Int64Counter
	activeChains    metric.Int64UpDownCounter

	// Graph metrics
	graphExecutions metric.Int64Counter
	graphDuration   metric.Float64Histogram
	graphErrors     metric.Int64Counter
	activeGraphs    metric.Int64UpDownCounter
	graphNodes      metric.Int64Histogram

	// Workflow metrics
	workflowExecutions metric.Int64Counter
	workflowDuration   metric.Float64Histogram
	workflowErrors     metric.Int64Counter
	activeWorkflows    metric.Int64UpDownCounter

	// General metrics
	totalExecutions metric.Int64Counter
	totalErrors     metric.Int64Counter

	// Tracer for span creation
	tracer trace.Tracer
}

// NewMetrics creates a new metrics instance with OTEL instrumentation.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	m := &Metrics{tracer: tracer}

	var err error

	// Initialize chain metrics
	m.chainExecutions, err = meter.Int64Counter(
		"orchestration_chain_executions_total",
		metric.WithDescription("Total number of chain executions"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.chainDuration, err = meter.Float64Histogram(
		"orchestration_chain_duration_seconds",
		metric.WithDescription("Duration of chain executions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.chainErrors, err = meter.Int64Counter(
		"orchestration_chain_errors_total",
		metric.WithDescription("Total number of chain execution errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.activeChains, err = meter.Int64UpDownCounter(
		"orchestration_active_chains",
		metric.WithDescription("Number of currently active chains"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize graph metrics
	m.graphExecutions, err = meter.Int64Counter(
		"orchestration_graph_executions_total",
		metric.WithDescription("Total number of graph executions"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.graphDuration, err = meter.Float64Histogram(
		"orchestration_graph_duration_seconds",
		metric.WithDescription("Duration of graph executions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.graphErrors, err = meter.Int64Counter(
		"orchestration_graph_errors_total",
		metric.WithDescription("Total number of graph execution errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.activeGraphs, err = meter.Int64UpDownCounter(
		"orchestration_active_graphs",
		metric.WithDescription("Number of currently active graphs"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.graphNodes, err = meter.Int64Histogram(
		"orchestration_graph_nodes",
		metric.WithDescription("Number of nodes in executed graphs"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize workflow metrics
	m.workflowExecutions, err = meter.Int64Counter(
		"orchestration_workflow_executions_total",
		metric.WithDescription("Total number of workflow executions"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.workflowDuration, err = meter.Float64Histogram(
		"orchestration_workflow_duration_seconds",
		metric.WithDescription("Duration of workflow executions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.workflowErrors, err = meter.Int64Counter(
		"orchestration_workflow_errors_total",
		metric.WithDescription("Total number of workflow execution errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.activeWorkflows, err = meter.Int64UpDownCounter(
		"orchestration_active_workflows",
		metric.WithDescription("Number of currently active workflows"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize general metrics
	m.totalExecutions, err = meter.Int64Counter(
		"orchestration_executions_total",
		metric.WithDescription("Total number of orchestration executions"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.totalErrors, err = meter.Int64Counter(
		"orchestration_errors_total",
		metric.WithDescription("Total number of orchestration errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// RecordChainExecution records a chain execution.
func (m *Metrics) RecordChainExecution(ctx context.Context, duration time.Duration, success bool, chainName string) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("chain_name", chainName),
		attribute.Bool("success", success),
	}

	if m.chainExecutions != nil {
		m.chainExecutions.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.totalExecutions != nil {
		m.totalExecutions.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.chainDuration != nil {
		m.chainDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	if !success {
		if m.chainErrors != nil {
			m.chainErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
		if m.totalErrors != nil {
			m.totalErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	}
}

// RecordChainActive records active chain count changes.
func (m *Metrics) RecordChainActive(ctx context.Context, delta int64, chainName string) {
	if m == nil || m.activeChains == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("chain_name", chainName),
	}

	m.activeChains.Add(ctx, delta, metric.WithAttributes(attrs...))
}

// RecordGraphExecution records a graph execution.
func (m *Metrics) RecordGraphExecution(ctx context.Context, duration time.Duration, success bool, graphName string, nodeCount int) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("graph_name", graphName),
		attribute.Bool("success", success),
	}

	if m.graphExecutions != nil {
		m.graphExecutions.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.totalExecutions != nil {
		m.totalExecutions.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.graphDuration != nil {
		m.graphDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
	if m.graphNodes != nil {
		m.graphNodes.Record(ctx, int64(nodeCount), metric.WithAttributes(attrs...))
	}

	if !success {
		if m.graphErrors != nil {
			m.graphErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
		if m.totalErrors != nil {
			m.totalErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	}
}

// RecordGraphActive records active graph count changes.
func (m *Metrics) RecordGraphActive(ctx context.Context, delta int64, graphName string) {
	if m == nil || m.activeGraphs == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("graph_name", graphName),
	}

	m.activeGraphs.Add(ctx, delta, metric.WithAttributes(attrs...))
}

// RecordWorkflowExecution records a workflow execution.
func (m *Metrics) RecordWorkflowExecution(ctx context.Context, duration time.Duration, success bool, workflowName string) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("workflow_name", workflowName),
		attribute.Bool("success", success),
	}

	if m.workflowExecutions != nil {
		m.workflowExecutions.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.totalExecutions != nil {
		m.totalExecutions.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.workflowDuration != nil {
		m.workflowDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	if !success {
		if m.workflowErrors != nil {
			m.workflowErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
		if m.totalErrors != nil {
			m.totalErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	}
}

// RecordWorkflowActive records active workflow count changes.
func (m *Metrics) RecordWorkflowActive(ctx context.Context, delta int64, workflowName string) {
	if m == nil || m.activeWorkflows == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("workflow_name", workflowName),
	}

	m.activeWorkflows.Add(ctx, delta, metric.WithAttributes(attrs...))
}

// StartChainSpan starts a new span for chain execution.
//
//nolint:spancheck // Spans are intentionally returned for caller to manage lifecycle
func (m *Metrics) StartChainSpan(ctx context.Context, chainName, operation string) (context.Context, trace.Span) {
	if m.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := m.tracer.Start(ctx, "orchestration.chain."+operation,
		trace.WithAttributes(
			attribute.String("chain.name", chainName),
		),
	)
	return ctx, span
}

// StartGraphSpan starts a new span for graph execution.
//
//nolint:spancheck // Spans are intentionally returned for caller to manage lifecycle
func (m *Metrics) StartGraphSpan(ctx context.Context, graphName, operation string) (context.Context, trace.Span) {
	if m.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := m.tracer.Start(ctx, "orchestration.graph."+operation,
		trace.WithAttributes(
			attribute.String("graph.name", graphName),
		),
	)
	return ctx, span
}

// StartWorkflowSpan starts a new span for workflow execution.
//
//nolint:spancheck // Spans are intentionally returned for caller to manage lifecycle
func (m *Metrics) StartWorkflowSpan(ctx context.Context, workflowName, operation string) (context.Context, trace.Span) {
	if m.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := m.tracer.Start(ctx, "orchestration.workflow."+operation,
		trace.WithAttributes(
			attribute.String("workflow.name", workflowName),
		),
	)
	return ctx, span
}

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func InitMetrics(meter metric.Meter) {
	metricsOnce.Do(func() {
		tracer := otel.Tracer("beluga.orchestration")
		metrics, err := NewMetrics(meter, tracer)
		if err != nil {
			// Fallback to no-op metrics if initialization fails
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

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{}
}
