package memory

import (
	"context"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
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

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
// For Before* hooks and OnError, the first error returned short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeSave: hookutil.ComposeError2(h, func(hk Hooks) func(context.Context, schema.Message, schema.Message) error {
			return hk.BeforeSave
		}),
		AfterSave: hookutil.ComposeVoid3(h, func(hk Hooks) func(context.Context, schema.Message, schema.Message, error) {
			return hk.AfterSave
		}),
		BeforeLoad: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, string) error {
			return hk.BeforeLoad
		}),
		AfterLoad: hookutil.ComposeVoid3(h, func(hk Hooks) func(context.Context, string, []schema.Message, error) {
			return hk.AfterLoad
		}),
		BeforeSearch: hookutil.ComposeError2(h, func(hk Hooks) func(context.Context, string, int) error {
			return hk.BeforeSearch
		}),
		AfterSearch: hookutil.ComposeVoid4(h, func(hk Hooks) func(context.Context, string, int, []schema.Document, error) {
			return hk.AfterSearch
		}),
		BeforeClear: hookutil.ComposeError0(h, func(hk Hooks) func(context.Context) error {
			return hk.BeforeClear
		}),
		AfterClear: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, error) {
			return hk.AfterClear
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
	}
}
