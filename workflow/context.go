package workflow

import (
	"context"
	"iter"
	"time"
)

// WorkflowContext extends context.Context with workflow-specific operations.
// It provides deterministic execution by recording activity results and
// replaying them during recovery.
type WorkflowContext interface {
	context.Context

	// ExecuteActivity runs an activity within the workflow. Results are
	// recorded for replay during recovery.
	ExecuteActivity(fn ActivityFunc, input any, opts ...ActivityOption) (any, error)

	// ReceiveSignal returns an iterator that yields payloads delivered to
	// the named signal. Iteration ends when the workflow context is
	// canceled. Multiple calls with the same name are independent
	// subscribers and each receives every payload delivered after
	// subscription.
	ReceiveSignal(name string) iter.Seq2[any, error]

	// Sleep pauses the workflow for the given duration. Unlike time.Sleep,
	// this is recorded and replayed correctly during recovery.
	Sleep(d time.Duration) error
}
