package auth

import "context"

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
	return Hooks{
		OnAuthorize: func(ctx context.Context, subject string, permission Permission, resource string) error {
			for _, h := range hooks {
				if h.OnAuthorize != nil {
					if err := h.OnAuthorize(ctx, subject, permission, resource); err != nil {
						return err
					}
				}
			}
			return nil
		},
		OnAllow: func(ctx context.Context, subject string, permission Permission, resource string) {
			for _, h := range hooks {
				if h.OnAllow != nil {
					h.OnAllow(ctx, subject, permission, resource)
				}
			}
		},
		OnDeny: func(ctx context.Context, subject string, permission Permission, resource string) {
			for _, h := range hooks {
				if h.OnDeny != nil {
					h.OnDeny(ctx, subject, permission, resource)
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
