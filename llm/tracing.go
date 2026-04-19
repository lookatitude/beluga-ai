package llm

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/v2/o11y"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// WithTracing returns middleware that wraps a ChatModel with OTel spans
// following the GenAI semantic conventions. Each operation produces a span
// named "llm.<op>" carrying a gen_ai.operation.name attribute. Errors are
// recorded on the span and the status is set to StatusError on failure.
//
// Enable tracing by composing with other middleware:
//
//	model = llm.ApplyMiddleware(model, llm.WithTracing(), llm.WithHooks(h))
func WithTracing() Middleware {
	return func(next ChatModel) ChatModel {
		return &tracedModel{next: next}
	}
}

// tracedModel wraps a ChatModel and emits a span around each operation.
type tracedModel struct {
	next ChatModel
}

func (m *tracedModel) Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
	ctx, span := o11y.StartSpan(ctx, "llm.generate", o11y.Attrs{
		o11y.AttrOperationName: "llm.generate",
		o11y.AttrRequestModel:  m.next.ModelID(),
	})
	defer span.End()

	resp, err := m.next.Generate(ctx, msgs, opts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	if resp != nil {
		span.SetAttributes(o11y.Attrs{
			o11y.AttrResponseModel:   resp.ModelID,
			o11y.AttrInputTokens:     resp.Usage.InputTokens,
			o11y.AttrOutputTokens:    resp.Usage.OutputTokens,
			o11y.AttrReasoningTokens: resp.Usage.ReasoningTokens,
		})
	}
	span.SetStatus(o11y.StatusOK, "")
	return resp, nil
}

func (m *tracedModel) Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	ctx, span := o11y.StartSpan(ctx, "llm.stream", o11y.Attrs{
		o11y.AttrOperationName: "llm.stream",
		o11y.AttrRequestModel:  m.next.ModelID(),
	})

	inner := m.next.Stream(ctx, msgs, opts...)
	return func(yield func(schema.StreamChunk, error) bool) {
		defer span.End()
		chunkCount := 0
		for chunk, err := range inner {
			if err != nil {
				span.RecordError(err)
				span.SetStatus(o11y.StatusError, err.Error())
				yield(schema.StreamChunk{}, err)
				return
			}
			chunkCount++
			if !yield(chunk, nil) {
				span.SetAttributes(o11y.Attrs{"gen_ai.stream.chunks": chunkCount})
				span.SetStatus(o11y.StatusOK, "")
				return
			}
		}
		span.SetAttributes(o11y.Attrs{"gen_ai.stream.chunks": chunkCount})
		span.SetStatus(o11y.StatusOK, "")
	}
}

func (m *tracedModel) BindTools(tools []schema.ToolDefinition) ChatModel {
	return &tracedModel{next: m.next.BindTools(tools)}
}

func (m *tracedModel) ModelID() string { return m.next.ModelID() }

// Ensure tracedModel implements ChatModel at compile time.
var _ ChatModel = (*tracedModel)(nil)
