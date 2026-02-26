package orchestration

import "context"

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

func composeBeforeStep(hooks []Hooks) func(context.Context, string, any) error {
	return func(ctx context.Context, stepName string, input any) error {
		for _, h := range hooks {
			if h.BeforeStep != nil {
				if err := h.BeforeStep(ctx, stepName, input); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterStep(hooks []Hooks) func(context.Context, string, any, error) {
	return func(ctx context.Context, stepName string, output any, err error) {
		for _, h := range hooks {
			if h.AfterStep != nil {
				h.AfterStep(ctx, stepName, output, err)
			}
		}
	}
}

func composeOnBranch(hooks []Hooks) func(context.Context, string, string) error {
	return func(ctx context.Context, from, to string) error {
		for _, h := range hooks {
			if h.OnBranch != nil {
				if err := h.OnBranch(ctx, from, to); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeOnError(hooks []Hooks) func(context.Context, error) error {
	return func(ctx context.Context, err error) error {
		for _, h := range hooks {
			if h.OnError != nil {
				if e := h.OnError(ctx, err); e != nil {
					return e
				}
			}
		}
		return err
	}
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
// For BeforeStep, OnBranch, and OnError, the first error returned short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeStep: composeBeforeStep(h),
		AfterStep:  composeAfterStep(h),
		OnBranch:   composeOnBranch(h),
		OnError:    composeOnError(h),
	}
}
