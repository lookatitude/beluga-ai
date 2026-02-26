package llm

import (
	"context"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
	"github.com/lookatitude/beluga-ai/schema"
)

// Hooks provides optional callback functions that are invoked at various
// points during LLM operations. All fields are optional; nil hooks are
// skipped. Hooks are composable via ComposeHooks.
type Hooks struct {
	// BeforeGenerate is called before each Generate or Stream call with the
	// input messages. Returning an error aborts the call.
	BeforeGenerate func(ctx context.Context, msgs []schema.Message) error

	// AfterGenerate is called after Generate completes with the response
	// and any error.
	AfterGenerate func(ctx context.Context, resp *schema.AIMessage, err error)

	// OnStream is called for each StreamChunk received during streaming.
	OnStream func(ctx context.Context, chunk schema.StreamChunk)

	// OnToolCall is called when the model produces a tool call.
	OnToolCall func(ctx context.Context, call schema.ToolCall)

	// OnError is called when an error occurs. The returned error replaces the
	// original; returning nil suppresses the error.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
// For BeforeGenerate and OnError, the first error returned short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeGenerate: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, []schema.Message) error {
			return hk.BeforeGenerate
		}),
		AfterGenerate: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, *schema.AIMessage, error) {
			return hk.AfterGenerate
		}),
		OnStream: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, schema.StreamChunk) {
			return hk.OnStream
		}),
		OnToolCall: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, schema.ToolCall) {
			return hk.OnToolCall
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
	}
}
