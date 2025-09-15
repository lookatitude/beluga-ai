package workflow

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"go.temporal.io/sdk/client"
)

// TemporalWorkflow implements the Workflow interface using Temporal
type TemporalWorkflow struct {
	config     iface.WorkflowConfig
	client     client.Client
	tracer     trace.Tracer
	workflowFn any
}

// NewTemporalWorkflow creates a new TemporalWorkflow
func NewTemporalWorkflow(config iface.WorkflowConfig, client client.Client, tracer trace.Tracer, workflowFn any) *TemporalWorkflow {
	return &TemporalWorkflow{
		config:     config,
		client:     client,
		tracer:     tracer,
		workflowFn: workflowFn,
	}
}

func (tw *TemporalWorkflow) Execute(ctx context.Context, input any) (string, string, error) {
	ctx, span := tw.tracer.Start(ctx, "temporal_workflow.execute",
		trace.WithAttributes(
			attribute.String("workflow.name", tw.config.Name),
			attribute.String("workflow.task_queue", tw.config.TaskQueue),
		))
	defer span.End()

	options := client.StartWorkflowOptions{
		ID:                       tw.config.Name,
		TaskQueue:                tw.config.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(tw.config.Timeout) * time.Second,
		// Note: RetryPolicy configuration would go here when Temporal client is available
	}

	we, err := tw.client.ExecuteWorkflow(ctx, options, tw.workflowFn, input)
	if err != nil {
		span.RecordError(err)
		return "", "", iface.ErrExecutionFailed("temporal_workflow.execute", err)
	}

	span.SetAttributes(
		attribute.String("workflow.id", we.GetID()),
		attribute.String("workflow.run_id", we.GetRunID()),
	)

	return we.GetID(), we.GetRunID(), nil
}

func (tw *TemporalWorkflow) GetResult(ctx context.Context, workflowID string, runID string) (any, error) {
	ctx, span := tw.tracer.Start(ctx, "temporal_workflow.get_result",
		trace.WithAttributes(
			attribute.String("workflow.id", workflowID),
			attribute.String("workflow.run_id", runID),
		))
	defer span.End()

	we := tw.client.GetWorkflow(ctx, workflowID, runID)
	var result any
	err := we.Get(ctx, &result)
	if err != nil {
		span.RecordError(err)
		return nil, iface.ErrExecutionFailed("temporal_workflow.get_result", err)
	}

	return result, nil
}

func (tw *TemporalWorkflow) Signal(ctx context.Context, workflowID string, runID string, signalName string, data any) error {
	ctx, span := tw.tracer.Start(ctx, "temporal_workflow.signal",
		trace.WithAttributes(
			attribute.String("workflow.id", workflowID),
			attribute.String("workflow.run_id", runID),
			attribute.String("signal.name", signalName),
		))
	defer span.End()

	err := tw.client.SignalWorkflow(ctx, workflowID, runID, signalName, data)
	if err != nil {
		span.RecordError(err)
		return iface.ErrExecutionFailed("temporal_workflow.signal", err)
	}

	return nil
}

func (tw *TemporalWorkflow) Query(ctx context.Context, workflowID string, runID string, queryType string, args ...any) (any, error) {
	ctx, span := tw.tracer.Start(ctx, "temporal_workflow.query",
		trace.WithAttributes(
			attribute.String("workflow.id", workflowID),
			attribute.String("workflow.run_id", runID),
			attribute.String("query.type", queryType),
		))
	defer span.End()

	value, err := tw.client.QueryWorkflow(ctx, workflowID, runID, queryType, args...)
	if err != nil {
		span.RecordError(err)
		return nil, iface.ErrExecutionFailed("temporal_workflow.query", err)
	}

	var result any
	if err := value.Get(&result); err != nil {
		return nil, iface.ErrExecutionFailed("temporal_workflow.query", fmt.Errorf("failed to decode query result: %w", err))
	}

	return result, nil
}

func (tw *TemporalWorkflow) Cancel(ctx context.Context, workflowID string, runID string) error {
	ctx, span := tw.tracer.Start(ctx, "temporal_workflow.cancel",
		trace.WithAttributes(
			attribute.String("workflow.id", workflowID),
			attribute.String("workflow.run_id", runID),
		))
	defer span.End()

	err := tw.client.CancelWorkflow(ctx, workflowID, runID)
	if err != nil {
		span.RecordError(err)
		return iface.ErrExecutionFailed("temporal_workflow.cancel", err)
	}

	return nil
}

func (tw *TemporalWorkflow) Terminate(ctx context.Context, workflowID string, runID string, reason string, details ...any) error {
	ctx, span := tw.tracer.Start(ctx, "temporal_workflow.terminate",
		trace.WithAttributes(
			attribute.String("workflow.id", workflowID),
			attribute.String("workflow.run_id", runID),
			attribute.String("terminate.reason", reason),
		))
	defer span.End()

	err := tw.client.TerminateWorkflow(ctx, workflowID, runID, reason, details...)
	if err != nil {
		span.RecordError(err)
		return iface.ErrExecutionFailed("temporal_workflow.terminate", err)
	}

	return nil
}

// TemporalActivityWrapper wraps a Beluga Runnable as a Temporal Activity
type TemporalActivityWrapper struct {
	Runnable core.Runnable
	tracer   trace.Tracer
}

// NewTemporalActivityWrapper creates a new activity wrapper
func NewTemporalActivityWrapper(runnable core.Runnable, tracer trace.Tracer) *TemporalActivityWrapper {
	return &TemporalActivityWrapper{
		Runnable: runnable,
		tracer:   tracer,
	}
}

// Execute runs the underlying Beluga Runnable as a Temporal Activity
func (w *TemporalActivityWrapper) Execute(ctx context.Context, input any) (any, error) {
	ctx, span := w.tracer.Start(ctx, "temporal_activity.execute",
		trace.WithAttributes(
			attribute.String("activity.type", fmt.Sprintf("%T", w.Runnable)),
		))
	defer span.End()

	output, err := w.Runnable.Invoke(ctx, input)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return output, nil
}

// RegisterActivities registers Beluga Runnables as Temporal Activities
// Note: This is a placeholder for Temporal activity registration
func RegisterActivities(runnables map[string]core.Runnable, tracer trace.Tracer) {
	for name, runnable := range runnables {
		wrapper := NewTemporalActivityWrapper(runnable, tracer)
		// Note: In a real implementation, you'd register this with the Temporal worker
		// This is a placeholder for the registration logic
		_ = wrapper
		_ = name
	}
}

// Ensure TemporalWorkflow implements the Workflow interface
var _ iface.Workflow = (*TemporalWorkflow)(nil)
