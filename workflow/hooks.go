package workflow

import (
	"context"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
)

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

// ComposeHooks merges multiple Hooks into one. Callbacks are called in order.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		OnWorkflowStart: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, string, any) {
			return hk.OnWorkflowStart
		}),
		OnWorkflowComplete: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, string, any) {
			return hk.OnWorkflowComplete
		}),
		OnWorkflowFail: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, string, error) {
			return hk.OnWorkflowFail
		}),
		OnActivityStart: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, string, any) {
			return hk.OnActivityStart
		}),
		OnActivityComplete: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, string, any) {
			return hk.OnActivityComplete
		}),
		OnSignal: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, string, Signal) {
			return hk.OnSignal
		}),
		OnRetry: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, string, error) {
			return hk.OnRetry
		}),
	}
}
