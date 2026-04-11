package workflow

import (
	"context"

	"github.com/lookatitude/beluga-ai/o11y"
)

// WithTracing returns a Middleware that wraps a DurableExecutor with OTel
// spans following the GenAI semantic conventions. Each operation produces a
// span named "workflow.<op>" carrying a gen_ai.operation.name attribute.
// Errors are recorded on the span and the status is set to StatusError on
// failure.
//
// Enable tracing by composing with other middleware:
//
//	exec = workflow.ApplyMiddleware(exec, workflow.WithTracing(), workflow.WithHooks(h))
func WithTracing() Middleware {
	return func(next DurableExecutor) DurableExecutor {
		return &tracedExecutor{next: next}
	}
}

// tracedExecutor wraps a DurableExecutor and emits a span around each operation.
type tracedExecutor struct {
	next DurableExecutor
}

func (e *tracedExecutor) Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error) {
	ctx, span := o11y.StartSpan(ctx, "workflow.execute", o11y.Attrs{
		o11y.AttrOperationName: "workflow.execute",
		"workflow.id":          opts.ID,
	})
	defer span.End()

	handle, err := e.next.Execute(ctx, fn, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetStatus(o11y.StatusOK, "")
	return handle, nil
}

func (e *tracedExecutor) Signal(ctx context.Context, workflowID string, signal Signal) error {
	ctx, span := o11y.StartSpan(ctx, "workflow.signal", o11y.Attrs{
		o11y.AttrOperationName: "workflow.signal",
		"workflow.id":          workflowID,
		"workflow.signal.name": signal.Name,
	})
	defer span.End()

	if err := e.next.Signal(ctx, workflowID, signal); err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

func (e *tracedExecutor) Query(ctx context.Context, workflowID string, queryType string) (any, error) {
	ctx, span := o11y.StartSpan(ctx, "workflow.query", o11y.Attrs{
		o11y.AttrOperationName: "workflow.query",
		"workflow.id":          workflowID,
		"workflow.query.type":  queryType,
	})
	defer span.End()

	result, err := e.next.Query(ctx, workflowID, queryType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetStatus(o11y.StatusOK, "")
	return result, nil
}

func (e *tracedExecutor) Cancel(ctx context.Context, workflowID string) error {
	ctx, span := o11y.StartSpan(ctx, "workflow.cancel", o11y.Attrs{
		o11y.AttrOperationName: "workflow.cancel",
		"workflow.id":          workflowID,
	})
	defer span.End()

	if err := e.next.Cancel(ctx, workflowID); err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

// Ensure tracedExecutor implements DurableExecutor at compile time.
var _ DurableExecutor = (*tracedExecutor)(nil)
