package workflow

import "context"

// Middleware wraps a DurableExecutor to add cross-cutting behavior.
type Middleware func(DurableExecutor) DurableExecutor

// ApplyMiddleware wraps executor with middlewares in reverse order so the first
// middleware is outermost (first to execute).
func ApplyMiddleware(e DurableExecutor, mws ...Middleware) DurableExecutor {
	for i := len(mws) - 1; i >= 0; i-- {
		e = mws[i](e)
	}
	return e
}

// WithHooks returns a Middleware that adds hooks to a DurableExecutor.
func WithHooks(h Hooks) Middleware {
	return func(next DurableExecutor) DurableExecutor {
		return &hookedExecutor{next: next, hooks: h}
	}
}

type hookedExecutor struct {
	next  DurableExecutor
	hooks Hooks
}

func (e *hookedExecutor) Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error) {
	if e.hooks.OnWorkflowStart != nil {
		e.hooks.OnWorkflowStart(ctx, opts.ID, opts.Input)
	}
	return e.next.Execute(ctx, fn, opts)
}

func (e *hookedExecutor) Signal(ctx context.Context, workflowID string, signal Signal) error {
	if e.hooks.OnSignal != nil {
		e.hooks.OnSignal(ctx, workflowID, signal)
	}
	return e.next.Signal(ctx, workflowID, signal)
}

func (e *hookedExecutor) Query(ctx context.Context, workflowID string, queryType string) (any, error) {
	return e.next.Query(ctx, workflowID, queryType)
}

func (e *hookedExecutor) Cancel(ctx context.Context, workflowID string) error {
	return e.next.Cancel(ctx, workflowID)
}

var _ DurableExecutor = (*hookedExecutor)(nil)
