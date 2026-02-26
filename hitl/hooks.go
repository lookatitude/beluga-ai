package hitl

import (
	"context"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
)

// Hooks provides lifecycle callbacks for the HITL manager.
// All fields are optional; nil hooks are skipped.
type Hooks struct {
	// OnRequest is called when a new interaction request is created.
	// Returning an error aborts the request.
	OnRequest func(ctx context.Context, req InteractionRequest) error

	// OnApprove is called when a request is approved (either auto or human).
	OnApprove func(ctx context.Context, req InteractionRequest, resp InteractionResponse)

	// OnReject is called when a request is rejected by a human.
	OnReject func(ctx context.Context, req InteractionRequest, resp InteractionResponse)

	// OnTimeout is called when a request times out without a response.
	OnTimeout func(ctx context.Context, req InteractionRequest)

	// OnError is called when an error occurs. The returned error replaces the
	// original; returning nil suppresses the error. A non-nil return
	// short-circuits composed hooks.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into one. Callbacks are called in the
// order the hooks were provided. For OnRequest and OnError, the first non-nil
// error return short-circuits (stops further hooks). OnError returns the
// original error if all hooks return nil.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		OnRequest: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, InteractionRequest) error {
			return hk.OnRequest
		}),
		OnApprove: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, InteractionRequest, InteractionResponse) {
			return hk.OnApprove
		}),
		OnReject: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, InteractionRequest, InteractionResponse) {
			return hk.OnReject
		}),
		OnTimeout: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, InteractionRequest) {
			return hk.OnTimeout
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
	}
}
