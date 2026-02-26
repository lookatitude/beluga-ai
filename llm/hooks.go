package llm

import (
	"context"

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

func composeBeforeGenerate(hooks []Hooks) func(context.Context, []schema.Message) error {
	return func(ctx context.Context, msgs []schema.Message) error {
		for _, h := range hooks {
			if h.BeforeGenerate != nil {
				if err := h.BeforeGenerate(ctx, msgs); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterGenerate(hooks []Hooks) func(context.Context, *schema.AIMessage, error) {
	return func(ctx context.Context, resp *schema.AIMessage, err error) {
		for _, h := range hooks {
			if h.AfterGenerate != nil {
				h.AfterGenerate(ctx, resp, err)
			}
		}
	}
}

func composeOnStream(hooks []Hooks) func(context.Context, schema.StreamChunk) {
	return func(ctx context.Context, chunk schema.StreamChunk) {
		for _, h := range hooks {
			if h.OnStream != nil {
				h.OnStream(ctx, chunk)
			}
		}
	}
}

func composeOnToolCall(hooks []Hooks) func(context.Context, schema.ToolCall) {
	return func(ctx context.Context, call schema.ToolCall) {
		for _, h := range hooks {
			if h.OnToolCall != nil {
				h.OnToolCall(ctx, call)
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
// For BeforeGenerate and OnError, the first error returned short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeGenerate: composeBeforeGenerate(h),
		AfterGenerate:  composeAfterGenerate(h),
		OnStream:       composeOnStream(h),
		OnToolCall:     composeOnToolCall(h),
		OnError:        composeOnError(h),
	}
}
