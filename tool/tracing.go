package tool

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/o11y"
)

// WithTracing returns middleware that wraps a Tool with OTel spans following
// the GenAI semantic conventions. Each Execute produces a span named
// "tool.execute" carrying a gen_ai.operation.name attribute and the tool
// name. Errors are recorded on the span and the status is set to StatusError
// on failure.
//
// Enable tracing by composing with other middleware:
//
//	t = tool.ApplyMiddleware(t, tool.WithTracing(), tool.WithRetry(3))
func WithTracing() Middleware {
	return func(next Tool) Tool {
		return &tracedTool{next: next}
	}
}

// tracedTool wraps a Tool and emits a span around Execute.
type tracedTool struct {
	next Tool
}

func (t *tracedTool) Name() string                { return t.next.Name() }
func (t *tracedTool) Description() string         { return t.next.Description() }
func (t *tracedTool) InputSchema() map[string]any { return t.next.InputSchema() }

func (t *tracedTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
	ctx, span := o11y.StartSpan(ctx, "tool.execute", o11y.Attrs{
		o11y.AttrOperationName: "tool.execute",
		o11y.AttrToolName:      t.next.Name(),
	})
	defer span.End()

	result, err := t.next.Execute(ctx, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	if result != nil {
		span.SetAttributes(o11y.Attrs{
			"tool.execute.is_error":      result.IsError,
			"tool.execute.content_parts": len(result.Content),
		})
		if result.IsError {
			span.SetStatus(o11y.StatusError, "tool returned is_error=true")
			return result, nil
		}
	}
	span.SetStatus(o11y.StatusOK, "")
	return result, nil
}

// Ensure tracedTool implements Tool at compile time.
var _ Tool = (*tracedTool)(nil)
