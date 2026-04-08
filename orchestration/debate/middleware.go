package debate

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/orchestration"
)

// Middleware wraps a core.Runnable to add cross-cutting behavior specific
// to debate and generator-evaluator patterns.
type Middleware func(core.Runnable) core.Runnable

// ApplyMiddleware wraps r with the given middlewares. The first middleware
// in the list is the outermost (first to execute).
func ApplyMiddleware(r core.Runnable, mws ...Middleware) core.Runnable {
	for i := len(mws) - 1; i >= 0; i-- {
		r = mws[i](r)
	}
	return r
}

// AsOrchestrationPattern wraps a core.Runnable with debate middleware and
// returns it as an OrchestrationPattern.
func AsOrchestrationPattern(name string, r core.Runnable, mws ...Middleware) orchestration.OrchestrationPattern {
	wrapped := ApplyMiddleware(r, mws...)
	return &middlewarePatternAdapter{
		name:    name,
		wrapped: wrapped,
	}
}

// middlewarePatternAdapter satisfies OrchestrationPattern for middleware-wrapped runnables.
type middlewarePatternAdapter struct {
	name    string
	wrapped core.Runnable
}

// Compile-time check.
var _ orchestration.OrchestrationPattern = (*middlewarePatternAdapter)(nil)

// Name returns the pattern name.
func (a *middlewarePatternAdapter) Name() string { return a.name }

// Invoke delegates to the middleware-wrapped runnable.
func (a *middlewarePatternAdapter) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	return a.wrapped.Invoke(ctx, input, opts...)
}

// Stream delegates to the middleware-wrapped runnable.
func (a *middlewarePatternAdapter) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return a.wrapped.Stream(ctx, input, opts...)
}
