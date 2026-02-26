// Package hookutil provides generic helpers for composing hook functions.
// Each helper takes a slice of hook structs and a field-extractor function,
// then returns a composed function that calls every non-nil hook in order.
//
// Usage pattern:
//
//	func composeOnStart(hooks []Hooks) func(context.Context, string) error {
//	    return hookutil.ComposeError1(hooks, func(h Hooks) func(context.Context, string) error {
//	        return h.OnStart
//	    })
//	}
//
// For the special OnError passthrough pattern — where the original error is
// returned if no hook replaces it — use ComposeErrorPassthrough.
package hookutil

import "context"

// ComposeErrorPassthrough composes hooks of the form func(context.Context, error) error.
// Each non-nil hook is called in order. The first hook that returns a non-nil
// error short-circuits and that error is returned. If all hooks return nil the
// original error is returned unchanged (passthrough semantics).
func ComposeErrorPassthrough[H any](hooks []H, get func(H) func(context.Context, error) error) func(context.Context, error) error {
	return func(ctx context.Context, err error) error {
		for _, h := range hooks {
			if fn := get(h); fn != nil {
				if e := fn(ctx, err); e != nil {
					return e
				}
			}
		}
		return err
	}
}

// ComposeErrorPassthrough1 composes hooks of the form func(context.Context, A, error) error.
// Each non-nil hook is called in order. The first hook that returns a non-nil
// error short-circuits and that error is returned. If all hooks return nil the
// original error is returned unchanged (passthrough semantics).
func ComposeErrorPassthrough1[H, A any](hooks []H, get func(H) func(context.Context, A, error) error) func(context.Context, A, error) error {
	return func(ctx context.Context, a A, err error) error {
		for _, h := range hooks {
			if fn := get(h); fn != nil {
				if e := fn(ctx, a, err); e != nil {
					return e
				}
			}
		}
		return err
	}
}

// ComposeError0 composes hooks of the form func(context.Context) error.
// Non-nil hooks are called in order; the first error short-circuits.
func ComposeError0[H any](hooks []H, get func(H) func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		for _, h := range hooks {
			if fn := get(h); fn != nil {
				if err := fn(ctx); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// ComposeError1 composes hooks of the form func(context.Context, A) error.
// Non-nil hooks are called in order; the first error short-circuits.
func ComposeError1[H, A any](hooks []H, get func(H) func(context.Context, A) error) func(context.Context, A) error {
	return func(ctx context.Context, a A) error {
		for _, h := range hooks {
			if fn := get(h); fn != nil {
				if err := fn(ctx, a); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// ComposeError2 composes hooks of the form func(context.Context, A, B) error.
// Non-nil hooks are called in order; the first error short-circuits.
func ComposeError2[H, A, B any](hooks []H, get func(H) func(context.Context, A, B) error) func(context.Context, A, B) error {
	return func(ctx context.Context, a A, b B) error {
		for _, h := range hooks {
			if fn := get(h); fn != nil {
				if err := fn(ctx, a, b); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// ComposeError3 composes hooks of the form func(context.Context, A, B, C) error.
// Non-nil hooks are called in order; the first error short-circuits.
func ComposeError3[H, A, B, C any](hooks []H, get func(H) func(context.Context, A, B, C) error) func(context.Context, A, B, C) error {
	return func(ctx context.Context, a A, b B, c C) error {
		for _, h := range hooks {
			if fn := get(h); fn != nil {
				if err := fn(ctx, a, b, c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// ComposeVoid0 composes void hooks of the form func(context.Context).
// All non-nil hooks are called in order unconditionally.
func ComposeVoid0[H any](hooks []H, get func(H) func(context.Context)) func(context.Context) {
	return func(ctx context.Context) {
		for _, h := range hooks {
			if fn := get(h); fn != nil {
				fn(ctx)
			}
		}
	}
}

// ComposeVoid1 composes void hooks of the form func(context.Context, A).
// All non-nil hooks are called in order unconditionally.
func ComposeVoid1[H, A any](hooks []H, get func(H) func(context.Context, A)) func(context.Context, A) {
	return func(ctx context.Context, a A) {
		for _, h := range hooks {
			if fn := get(h); fn != nil {
				fn(ctx, a)
			}
		}
	}
}

// ComposeVoid2 composes void hooks of the form func(context.Context, A, B).
// All non-nil hooks are called in order unconditionally.
func ComposeVoid2[H, A, B any](hooks []H, get func(H) func(context.Context, A, B)) func(context.Context, A, B) {
	return func(ctx context.Context, a A, b B) {
		for _, h := range hooks {
			if fn := get(h); fn != nil {
				fn(ctx, a, b)
			}
		}
	}
}

// ComposeVoid3 composes void hooks of the form func(context.Context, A, B, C).
// All non-nil hooks are called in order unconditionally.
func ComposeVoid3[H, A, B, C any](hooks []H, get func(H) func(context.Context, A, B, C)) func(context.Context, A, B, C) {
	return func(ctx context.Context, a A, b B, c C) {
		for _, h := range hooks {
			if fn := get(h); fn != nil {
				fn(ctx, a, b, c)
			}
		}
	}
}

// ComposeVoid4 composes void hooks of the form func(context.Context, A, B, C, D).
// All non-nil hooks are called in order unconditionally.
func ComposeVoid4[H, A, B, C, D any](hooks []H, get func(H) func(context.Context, A, B, C, D)) func(context.Context, A, B, C, D) {
	return func(ctx context.Context, a A, b B, c C, d D) {
		for _, h := range hooks {
			if fn := get(h); fn != nil {
				fn(ctx, a, b, c, d)
			}
		}
	}
}
