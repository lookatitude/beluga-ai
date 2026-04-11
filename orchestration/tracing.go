package orchestration

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
)

// WithTracing returns middleware that wraps a core.Runnable with OTel spans
// following the GenAI semantic conventions. Invoke and Stream each produce a
// span ("orchestration.invoke" / "orchestration.stream") carrying a
// gen_ai.operation.name attribute. Errors are recorded on the span and the
// status is set to StatusError on failure.
//
// Enable tracing by composing with other middleware:
//
//	r = orchestration.ApplyMiddleware(r, orchestration.WithTracing())
func WithTracing() Middleware {
	return func(next core.Runnable) core.Runnable {
		return &tracedRunnable{next: next}
	}
}

// tracedRunnable wraps a core.Runnable and emits a span around each call.
type tracedRunnable struct {
	next core.Runnable
}

func (r *tracedRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	ctx, span := o11y.StartSpan(ctx, "orchestration.invoke", o11y.Attrs{
		o11y.AttrOperationName: "orchestration.invoke",
	})
	defer span.End()

	out, err := r.next.Invoke(ctx, input, opts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetStatus(o11y.StatusOK, "")
	return out, nil
}

func (r *tracedRunnable) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		ctx, span := o11y.StartSpan(ctx, "orchestration.stream", o11y.Attrs{
			o11y.AttrOperationName: "orchestration.stream",
		})
		defer span.End()

		var sawErr bool
		for v, err := range r.next.Stream(ctx, input, opts...) {
			if err != nil {
				sawErr = true
				span.RecordError(err)
				span.SetStatus(o11y.StatusError, err.Error())
			}
			if !yield(v, err) {
				return
			}
		}
		if !sawErr {
			span.SetStatus(o11y.StatusOK, "")
		}
	}
}

// Ensure tracedRunnable implements core.Runnable at compile time.
var _ core.Runnable = (*tracedRunnable)(nil)
