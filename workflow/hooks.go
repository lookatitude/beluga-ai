package workflow

import "context"

// Hooks provides lifecycle callbacks for the workflow executor.
// All fields are optional; nil hooks are skipped.
type Hooks struct {
	// OnWorkflowStart is called when a workflow begins execution.
	OnWorkflowStart func(ctx context.Context, workflowID string, input any)
	// OnWorkflowComplete is called when a workflow finishes successfully.
	OnWorkflowComplete func(ctx context.Context, workflowID string, result any)
	// OnWorkflowFail is called when a workflow fails.
	OnWorkflowFail func(ctx context.Context, workflowID string, err error)
	// OnActivityStart is called when an activity begins.
	OnActivityStart func(ctx context.Context, workflowID string, input any)
	// OnActivityComplete is called when an activity completes successfully.
	OnActivityComplete func(ctx context.Context, workflowID string, result any)
	// OnSignal is called when a signal is delivered to a workflow.
	OnSignal func(ctx context.Context, workflowID string, signal Signal)
	// OnRetry is called when an activity is retried.
	OnRetry func(ctx context.Context, workflowID string, err error)
}

func composeOnWorkflowStart(hooks []Hooks) func(context.Context, string, any) {
	return func(ctx context.Context, wfID string, input any) {
		for _, h := range hooks {
			if h.OnWorkflowStart != nil {
				h.OnWorkflowStart(ctx, wfID, input)
			}
		}
	}
}

func composeOnWorkflowComplete(hooks []Hooks) func(context.Context, string, any) {
	return func(ctx context.Context, wfID string, result any) {
		for _, h := range hooks {
			if h.OnWorkflowComplete != nil {
				h.OnWorkflowComplete(ctx, wfID, result)
			}
		}
	}
}

func composeOnWorkflowFail(hooks []Hooks) func(context.Context, string, error) {
	return func(ctx context.Context, wfID string, err error) {
		for _, h := range hooks {
			if h.OnWorkflowFail != nil {
				h.OnWorkflowFail(ctx, wfID, err)
			}
		}
	}
}

func composeOnActivityStart(hooks []Hooks) func(context.Context, string, any) {
	return func(ctx context.Context, wfID string, input any) {
		for _, h := range hooks {
			if h.OnActivityStart != nil {
				h.OnActivityStart(ctx, wfID, input)
			}
		}
	}
}

func composeOnActivityComplete(hooks []Hooks) func(context.Context, string, any) {
	return func(ctx context.Context, wfID string, result any) {
		for _, h := range hooks {
			if h.OnActivityComplete != nil {
				h.OnActivityComplete(ctx, wfID, result)
			}
		}
	}
}

func composeOnSignal(hooks []Hooks) func(context.Context, string, Signal) {
	return func(ctx context.Context, wfID string, signal Signal) {
		for _, h := range hooks {
			if h.OnSignal != nil {
				h.OnSignal(ctx, wfID, signal)
			}
		}
	}
}

func composeOnRetry(hooks []Hooks) func(context.Context, string, error) {
	return func(ctx context.Context, wfID string, err error) {
		for _, h := range hooks {
			if h.OnRetry != nil {
				h.OnRetry(ctx, wfID, err)
			}
		}
	}
}

// ComposeHooks merges multiple Hooks into one. Callbacks are called in order.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		OnWorkflowStart:    composeOnWorkflowStart(h),
		OnWorkflowComplete: composeOnWorkflowComplete(h),
		OnWorkflowFail:     composeOnWorkflowFail(h),
		OnActivityStart:    composeOnActivityStart(h),
		OnActivityComplete: composeOnActivityComplete(h),
		OnSignal:           composeOnSignal(h),
		OnRetry:            composeOnRetry(h),
	}
}
