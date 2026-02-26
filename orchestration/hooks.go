package orchestration

import (
	"context"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
)

// Hooks provides optional callback functions invoked at various points during
// orchestration execution. All fields are optional; nil hooks are skipped.
// Hooks are composable via ComposeHooks.
type Hooks struct {
	// BeforeStep is called before a named step begins execution.
	// Returning an error aborts the step.
	BeforeStep func(ctx context.Context, stepName string, input any) error

	// AfterStep is called after a named step completes.
	AfterStep func(ctx context.Context, stepName string, output any, err error)

	// OnBranch is called when traversal moves from one node to another.
	// Returning an error aborts the traversal.
	OnBranch func(ctx context.Context, from, to string) error

	// OnError is called when an error occurs. The returned error replaces the
	// original; returning nil suppresses the error. A non-nil return
	// short-circuits further hook processing.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
// For BeforeStep, OnBranch, and OnError, the first error returned short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeStep: hookutil.ComposeError2(h, func(hk Hooks) func(context.Context, string, any) error {
			return hk.BeforeStep
		}),
		AfterStep: hookutil.ComposeVoid3(h, func(hk Hooks) func(context.Context, string, any, error) {
			return hk.AfterStep
		}),
		OnBranch: hookutil.ComposeError2(h, func(hk Hooks) func(context.Context, string, string) error {
			return hk.OnBranch
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
	}
}
