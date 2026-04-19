package orchestration

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// OrchestrationPattern is the extension interface for named orchestration
// patterns. Every pattern that participates in the orchestration registry must
// implement this interface.
//
// OrchestrationPattern embeds [core.Runnable] so it can be composed
// transparently with the rest of the framework. The additional Name method
// identifies the pattern for observability and registry lookup.
type OrchestrationPattern interface {
	core.Runnable

	// Name returns the human-readable identifier for this pattern.
	Name() string
}

// patternAdapter wraps any core.Runnable together with a name to satisfy
// OrchestrationPattern. This is a convenience for tests and simple wrappers.
type patternAdapter struct {
	name   string
	invoke func(ctx context.Context, input any, opts ...core.Option) (any, error)
	stream func(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error]
}

// Name returns the pattern name.
func (p *patternAdapter) Name() string { return p.name }

// Invoke delegates to the underlying invoke function.
func (p *patternAdapter) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	return p.invoke(ctx, input, opts...)
}

// Stream delegates to the underlying stream function.
func (p *patternAdapter) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return p.stream(ctx, input, opts...)
}
