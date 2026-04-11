package routing

import (
	"context"

	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/schema"
)

// WithTracing returns middleware that wraps a CostRouter with OTel spans
// following the GenAI semantic conventions. SelectModel produces a span
// named "llm.routing.select_model" carrying a gen_ai.operation.name
// attribute. Errors are recorded on the span and the status is set to
// StatusError on failure.
//
// Enable tracing by composing with other middleware:
//
//	router = routing.ApplyMiddleware(router, routing.WithTracing())
func WithTracing() Middleware {
	return func(next CostRouter) CostRouter {
		return &tracedRouter{next: next}
	}
}

// tracedRouter wraps a CostRouter and emits a span around each operation.
type tracedRouter struct {
	next CostRouter
}

func (r *tracedRouter) SelectModel(ctx context.Context, msgs []schema.Message) (ModelSelection, error) {
	ctx, span := o11y.StartSpan(ctx, "llm.routing.select_model", o11y.Attrs{
		o11y.AttrOperationName: "llm.routing.select_model",
	})
	defer span.End()

	sel, err := r.next.SelectModel(ctx, msgs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return ModelSelection{}, err
	}
	span.SetAttributes(o11y.Attrs{
		o11y.AttrRequestModel: sel.ModelID,
	})
	span.SetStatus(o11y.StatusOK, "")
	return sel, nil
}

// Ensure tracedRouter implements CostRouter at compile time.
var _ CostRouter = (*tracedRouter)(nil)
