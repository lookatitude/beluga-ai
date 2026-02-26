package server

import (
	"context"
	"net/http"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
)

// Hooks provides optional callback functions that are invoked at various
// points during HTTP request processing. All fields are optional; nil hooks
// are skipped. Hooks are composable via ComposeHooks.
type Hooks struct {
	// BeforeRequest is called before request processing. Returning an error
	// aborts the request with a 500 status.
	BeforeRequest func(ctx context.Context, r *http.Request) error

	// AfterRequest is called after request processing completes with the
	// response status code.
	AfterRequest func(ctx context.Context, r *http.Request, statusCode int)

	// OnError is called when an error occurs. The returned error replaces the
	// original; returning nil suppresses the error. A non-nil return
	// short-circuits when composing multiple hooks.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks value. Callbacks are
// called in the order the hooks were provided. For BeforeRequest and OnError,
// the first error returned short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeRequest: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, *http.Request) error {
			return hk.BeforeRequest
		}),
		AfterRequest: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, *http.Request, int) {
			return hk.AfterRequest
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
	}
}
