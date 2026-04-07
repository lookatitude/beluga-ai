package plugins

import (
	"context"
	"sync/atomic"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/runtime"
	"github.com/lookatitude/beluga-ai/schema"
)

// Compile-time check that retryPlugin satisfies runtime.Plugin.
var _ runtime.Plugin = (*retryPlugin)(nil)

// retryPlugin retries retryable errors up to a configurable maximum.
type retryPlugin struct {
	maxRetries int
	count      atomic.Int64
}

// NewRetryAndReflect creates a Plugin that suppresses retryable errors (as
// determined by [core.IsRetryable]) up to maxRetries times, allowing the
// runner to attempt another turn. Once maxRetries is exhausted the original
// error is returned unchanged. Non-retryable errors are always forwarded
// immediately.
//
// WARNING: The retry counter is global across all sessions. In multi-tenant
// deployments, consider implementing per-session retry tracking to prevent one
// tenant's retries from consuming budget allocated for other tenants. For
// per-session limiting, create one plugin instance per session.
//
// The internal retry counter is per-plugin-instance and accumulates across
// all sessions served by the same Runner. For per-session limiting create
// one plugin per session.
func NewRetryAndReflect(maxRetries int) runtime.Plugin {
	return &retryPlugin{maxRetries: maxRetries}
}

// Name returns the plugin identifier.
func (p *retryPlugin) Name() string { return "retry_reflect" }

// BeforeTurn is a no-op for this plugin.
func (p *retryPlugin) BeforeTurn(_ context.Context, _ *runtime.Session, input schema.Message) (schema.Message, error) {
	return input, nil
}

// AfterTurn is a no-op for this plugin.
func (p *retryPlugin) AfterTurn(_ context.Context, _ *runtime.Session, events []agent.Event) ([]agent.Event, error) {
	return events, nil
}

// OnError suppresses the error and returns nil when the error is retryable and
// the retry budget has not been exhausted. In all other cases the original
// error is returned.
func (p *retryPlugin) OnError(_ context.Context, err error) error {
	if err == nil {
		return nil
	}
	if !core.IsRetryable(err) {
		return err
	}
	n := p.count.Add(1)
	if int(n) <= p.maxRetries {
		// Suppress the error; the runner will see nil and may retry.
		return nil
	}
	return err
}
