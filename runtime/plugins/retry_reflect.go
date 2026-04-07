package plugins

import (
	"context"
	"sync/atomic"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/runtime"
	"github.com/lookatitude/beluga-ai/schema"
)

// compile-time interface check.
var _ runtime.Plugin = (*retryAndReflect)(nil)

// retryAndReflect is a plugin that allows automatic retry of turns on
// retryable errors up to a configured maximum retry count.
type retryAndReflect struct {
	maxRetries int
	retries    atomic.Int32
}

// NewRetryAndReflect returns a runtime.Plugin that allows retrying a turn on
// retryable errors (as determined by core.IsRetryable) up to maxRetries times.
// The retry counter is shared across all turns in the plugin instance lifetime.
// Callers should create a new instance per session for isolated retry counts.
func NewRetryAndReflect(maxRetries int) runtime.Plugin {
	if maxRetries < 0 {
		maxRetries = 0
	}
	return &retryAndReflect{maxRetries: maxRetries}
}

// Name returns the plugin identifier.
func (r *retryAndReflect) Name() string { return "retry_reflect" }

// BeforeTurn is a no-op for this plugin; it passes the message through unchanged.
func (r *retryAndReflect) BeforeTurn(_ context.Context, _ *runtime.Session, input schema.Message) (schema.Message, error) {
	return input, nil
}

// AfterTurn is a no-op for this plugin; it passes events through unchanged.
func (r *retryAndReflect) AfterTurn(_ context.Context, _ *runtime.Session, events []agent.Event) ([]agent.Event, error) {
	return events, nil
}

// OnError checks whether err is retryable and whether the retry budget has not
// been exhausted. If both conditions are true, it increments the retry counter
// and returns nil to signal that the caller should retry. Otherwise it returns
// the original error unchanged.
func (r *retryAndReflect) OnError(_ context.Context, err error) error {
	if !core.IsRetryable(err) {
		return err
	}
	current := r.retries.Load()
	if int(current) >= r.maxRetries {
		return err
	}
	r.retries.Add(1)
	// Returning nil instructs the runtime to retry the turn.
	return nil
}
