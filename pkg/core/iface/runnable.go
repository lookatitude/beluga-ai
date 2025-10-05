// Package iface defines core interfaces for the Beluga AI Framework.
// T002: Move Runnable interface to iface/runnable.go while preserving existing imports
package iface

import (
	"context"
)

// Note: Option and OptionFunc are defined in option.go

// Runnable represents any component that can be executed in AI workflows.
// This is the foundational interface that enables unified orchestration
// of LLMs, retrievers, agents, chains, and other AI components.
type Runnable interface {
	// Invoke executes the runnable component with a single input and returns a single output.
	// Context MUST support cancellation and timeout.
	// Options MUST support functional configuration pattern.
	Invoke(ctx context.Context, input any, options ...Option) (any, error)

	// Batch executes the runnable component with multiple inputs concurrently or sequentially.
	// MUST handle partial failures gracefully and respect context cancellation.
	Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error)

	// Stream executes the runnable component with streaming output.
	// Channel MUST be closed when complete.
	// Errors MUST be sent through channel or returned immediately.
	Stream(ctx context.Context, input any, options ...Option) (<-chan any, error)
}
