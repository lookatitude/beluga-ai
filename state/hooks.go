package state

import "context"

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

func composeBeforeGet(hooks []Hooks) func(context.Context, string) error {
	return func(ctx context.Context, key string) error {
		for _, h := range hooks {
			if h.BeforeGet != nil {
				if err := h.BeforeGet(ctx, key); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterGet(hooks []Hooks) func(context.Context, string, any, error) {
	return func(ctx context.Context, key string, value any, err error) {
		for _, h := range hooks {
			if h.AfterGet != nil {
				h.AfterGet(ctx, key, value, err)
			}
		}
	}
}

func composeBeforeSet(hooks []Hooks) func(context.Context, string, any) error {
	return func(ctx context.Context, key string, value any) error {
		for _, h := range hooks {
			if h.BeforeSet != nil {
				if err := h.BeforeSet(ctx, key, value); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterSet(hooks []Hooks) func(context.Context, string, any, error) {
	return func(ctx context.Context, key string, value any, err error) {
		for _, h := range hooks {
			if h.AfterSet != nil {
				h.AfterSet(ctx, key, value, err)
			}
		}
	}
}

func composeOnDelete(hooks []Hooks) func(context.Context, string) error {
	return func(ctx context.Context, key string) error {
		for _, h := range hooks {
			if h.OnDelete != nil {
				if err := h.OnDelete(ctx, key); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeOnWatch(hooks []Hooks) func(context.Context, string) error {
	return func(ctx context.Context, key string) error {
		for _, h := range hooks {
			if h.OnWatch != nil {
				if err := h.OnWatch(ctx, key); err != nil {
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
// For Before* hooks, OnDelete, OnWatch, and OnError, the first error
// returned short-circuits (remaining hooks are not called).
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeGet: composeBeforeGet(h),
		AfterGet:  composeAfterGet(h),
		BeforeSet: composeBeforeSet(h),
		AfterSet:  composeAfterSet(h),
		OnDelete:  composeOnDelete(h),
		OnWatch:   composeOnWatch(h),
		OnError:   composeOnError(h),
	}
}
