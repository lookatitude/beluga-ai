package auth

import (
	"context"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
)

// Hooks provides optional callback functions for authorization operations.
// All fields are optional — nil hooks are skipped. Hooks can be composed
// via ComposeHooks.
type Hooks struct {
	// OnAuthorize is called before each authorization check. Returning an
	// error aborts the check.
	OnAuthorize func(ctx context.Context, subject string, permission Permission, resource string) error

	// OnAllow is called when authorization succeeds.
	OnAllow func(ctx context.Context, subject string, permission Permission, resource string)

	// OnDeny is called when authorization is denied.
	OnDeny func(ctx context.Context, subject string, permission Permission, resource string)

	// OnError is called when an error occurs. The returned error replaces the
	// original; returning nil suppresses the error. A non-nil return
	// short-circuits when composing multiple hooks.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
// For OnAuthorize and OnError, the first error returned short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		OnAuthorize: hookutil.ComposeError3(h, func(hk Hooks) func(context.Context, string, Permission, string) error {
			return hk.OnAuthorize
		}),
		OnAllow: hookutil.ComposeVoid3(h, func(hk Hooks) func(context.Context, string, Permission, string) {
			return hk.OnAllow
		}),
		OnDeny: hookutil.ComposeVoid3(h, func(hk Hooks) func(context.Context, string, Permission, string) {
			return hk.OnDeny
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
	}
}
