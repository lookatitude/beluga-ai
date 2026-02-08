package hitl

import "context"

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
	return Hooks{
		OnRequest: func(ctx context.Context, req InteractionRequest) error {
			for _, h := range hooks {
				if h.OnRequest != nil {
					if err := h.OnRequest(ctx, req); err != nil {
						return err
					}
				}
			}
			return nil
		},
		OnApprove: func(ctx context.Context, req InteractionRequest, resp InteractionResponse) {
			for _, h := range hooks {
				if h.OnApprove != nil {
					h.OnApprove(ctx, req, resp)
				}
			}
		},
		OnReject: func(ctx context.Context, req InteractionRequest, resp InteractionResponse) {
			for _, h := range hooks {
				if h.OnReject != nil {
					h.OnReject(ctx, req, resp)
				}
			}
		},
		OnTimeout: func(ctx context.Context, req InteractionRequest) {
			for _, h := range hooks {
				if h.OnTimeout != nil {
					h.OnTimeout(ctx, req)
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
