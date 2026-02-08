package workflow

import (
	"context"
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

	// ReceiveSignal returns a channel that delivers payloads for the named
	// signal. Multiple calls with the same name return the same channel.
	ReceiveSignal(name string) <-chan any

	// Sleep pauses the workflow for the given duration. Unlike time.Sleep,
	// this is recorded and replayed correctly during recovery.
	Sleep(d time.Duration) error
}
