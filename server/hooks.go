package server

import (
	"context"
	"net/http"
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
	return Hooks{
		BeforeRequest: func(ctx context.Context, r *http.Request) error {
			for _, h := range hooks {
				if h.BeforeRequest != nil {
					if err := h.BeforeRequest(ctx, r); err != nil {
						return err
					}
				}
			}
			return nil
		},
		AfterRequest: func(ctx context.Context, r *http.Request, statusCode int) {
			for _, h := range hooks {
				if h.AfterRequest != nil {
					h.AfterRequest(ctx, r, statusCode)
				}
			}
		},
		OnError: func(ctx context.Context, err error) error {
			for _, h := range hooks {
				if h.OnError != nil {
					if e := h.OnError(ctx, err); e != nil {
						return e
					}
				}
			}
			return err
		},
	}
}
