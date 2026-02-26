package memory

import (
	"context"

	"github.com/lookatitude/beluga-ai/schema"
)

// Hooks provides optional callback functions for Memory operations.
// All fields are optional — nil hooks are skipped. Hooks can be composed
// via ComposeHooks.
type Hooks struct {
	// BeforeSave is called before saving a message pair. Returning an error
	// aborts the save.
	BeforeSave func(ctx context.Context, input, output schema.Message) error

	// AfterSave is called after saving completes, with any error that occurred.
	AfterSave func(ctx context.Context, input, output schema.Message, err error)

	// BeforeLoad is called before loading messages. Returning an error aborts
	// the load.
	BeforeLoad func(ctx context.Context, query string) error

	// AfterLoad is called after loading completes, with the results and any
	// error that occurred.
	AfterLoad func(ctx context.Context, query string, msgs []schema.Message, err error)

	// BeforeSearch is called before searching for documents. Returning an
	// error aborts the search.
	BeforeSearch func(ctx context.Context, query string, k int) error

	// AfterSearch is called after searching completes, with the results and
	// any error that occurred.
	AfterSearch func(ctx context.Context, query string, k int, docs []schema.Document, err error)

	// BeforeClear is called before clearing memory. Returning an error aborts
	// the clear.
	BeforeClear func(ctx context.Context) error

	// AfterClear is called after clearing completes, with any error that occurred.
	AfterClear func(ctx context.Context, err error)

	// OnError is called when an error occurs. The returned error replaces the
	// original; returning nil suppresses the error.
	OnError func(ctx context.Context, err error) error
}

func composeBeforeSave(hooks []Hooks) func(context.Context, schema.Message, schema.Message) error {
	return func(ctx context.Context, input, output schema.Message) error {
		for _, h := range hooks {
			if h.BeforeSave != nil {
				if err := h.BeforeSave(ctx, input, output); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterSave(hooks []Hooks) func(context.Context, schema.Message, schema.Message, error) {
	return func(ctx context.Context, input, output schema.Message, err error) {
		for _, h := range hooks {
			if h.AfterSave != nil {
				h.AfterSave(ctx, input, output, err)
			}
		}
	}
}

func composeBeforeLoad(hooks []Hooks) func(context.Context, string) error {
	return func(ctx context.Context, query string) error {
		for _, h := range hooks {
			if h.BeforeLoad != nil {
				if err := h.BeforeLoad(ctx, query); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterLoad(hooks []Hooks) func(context.Context, string, []schema.Message, error) {
	return func(ctx context.Context, query string, msgs []schema.Message, err error) {
		for _, h := range hooks {
			if h.AfterLoad != nil {
				h.AfterLoad(ctx, query, msgs, err)
			}
		}
	}
}

func composeBeforeSearch(hooks []Hooks) func(context.Context, string, int) error {
	return func(ctx context.Context, query string, k int) error {
		for _, h := range hooks {
			if h.BeforeSearch != nil {
				if err := h.BeforeSearch(ctx, query, k); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterSearch(hooks []Hooks) func(context.Context, string, int, []schema.Document, error) {
	return func(ctx context.Context, query string, k int, docs []schema.Document, err error) {
		for _, h := range hooks {
			if h.AfterSearch != nil {
				h.AfterSearch(ctx, query, k, docs, err)
			}
		}
	}
}

func composeBeforeClear(hooks []Hooks) func(context.Context) error {
	return func(ctx context.Context) error {
		for _, h := range hooks {
			if h.BeforeClear != nil {
				if err := h.BeforeClear(ctx); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterClear(hooks []Hooks) func(context.Context, error) {
	return func(ctx context.Context, err error) {
		for _, h := range hooks {
			if h.AfterClear != nil {
				h.AfterClear(ctx, err)
			}
		}
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
// For Before* hooks and OnError, the first error returned short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeSave:   composeBeforeSave(h),
		AfterSave:    composeAfterSave(h),
		BeforeLoad:   composeBeforeLoad(h),
		AfterLoad:    composeAfterLoad(h),
		BeforeSearch: composeBeforeSearch(h),
		AfterSearch:  composeAfterSearch(h),
		BeforeClear:  composeBeforeClear(h),
		AfterClear:   composeAfterClear(h),
		OnError:      composeOnError(h),
	}
}
