package state

import (
	"context"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
)

// Hooks provides optional callback functions for Store operations.
// All fields are optional — nil hooks are skipped. Hooks can be composed
// via ComposeHooks.
type Hooks struct {
	// BeforeGet is called before retrieving a key. Returning an error
	// aborts the get.
	BeforeGet func(ctx context.Context, key string) error

	// AfterGet is called after a get completes, with the result and any error.
	AfterGet func(ctx context.Context, key string, value any, err error)

	// BeforeSet is called before storing a value. Returning an error
	// aborts the set.
	BeforeSet func(ctx context.Context, key string, value any) error

	// AfterSet is called after a set completes, with any error that occurred.
	AfterSet func(ctx context.Context, key string, value any, err error)

	// OnDelete is called before deleting a key. Returning an error aborts
	// the delete.
	OnDelete func(ctx context.Context, key string) error

	// OnWatch is called before establishing a watch. Returning an error
	// aborts the watch.
	OnWatch func(ctx context.Context, key string) error

	// OnError is called when an error occurs. The returned error replaces
	// the original; returning nil suppresses the error. A non-nil return
	// short-circuits when hooks are composed.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
// For Before* hooks, OnDelete, OnWatch, and OnError, the first error
// returned short-circuits (remaining hooks are not called).
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeGet: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, string) error {
			return hk.BeforeGet
		}),
		AfterGet: hookutil.ComposeVoid3(h, func(hk Hooks) func(context.Context, string, any, error) {
			return hk.AfterGet
		}),
		BeforeSet: hookutil.ComposeError2(h, func(hk Hooks) func(context.Context, string, any) error {
			return hk.BeforeSet
		}),
		AfterSet: hookutil.ComposeVoid3(h, func(hk Hooks) func(context.Context, string, any, error) {
			return hk.AfterSet
		}),
		OnDelete: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, string) error {
			return hk.OnDelete
		}),
		OnWatch: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, string) error {
			return hk.OnWatch
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
	}
}
