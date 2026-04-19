package sandbox

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/internal/hookutil"
)

// Hooks provides lifecycle callbacks for sandbox execution. All fields are
// optional — nil hooks are skipped. Hooks can be composed using ComposeHooks.
type Hooks struct {
	// BeforeExecute is called before code execution starts. It receives the
	// code and config. Returning an error aborts execution.
	BeforeExecute func(ctx context.Context, code string, cfg SandboxConfig) error

	// AfterExecute is called after code execution completes (successfully or
	// not). It receives the result and any error from execution.
	AfterExecute func(ctx context.Context, result ExecutionResult, err error)

	// OnTimeout is called when execution exceeds its deadline. It receives
	// the code and config of the timed-out execution.
	OnTimeout func(ctx context.Context, code string, cfg SandboxConfig)

	// OnError is called when execution fails with an error (not including
	// non-zero exit codes, which are reported via AfterExecute). Returning
	// a non-nil error propagates it; returning nil suppresses the original.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks struct.
// BeforeExecute hooks run in order; if any returns an error, subsequent hooks
// are skipped. AfterExecute and OnTimeout hooks run in order unconditionally.
// OnError hooks run in order; the first non-nil return wins.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeExecute: hookutil.ComposeError2(h, func(hk Hooks) func(context.Context, string, SandboxConfig) error {
			return hk.BeforeExecute
		}),
		AfterExecute: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, ExecutionResult, error) {
			return hk.AfterExecute
		}),
		OnTimeout: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, string, SandboxConfig) {
			return hk.OnTimeout
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
	}
}
