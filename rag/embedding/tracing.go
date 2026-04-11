package embedding

import (
	"context"

	"github.com/lookatitude/beluga-ai/o11y"
)

// WithTracing returns middleware that wraps an Embedder with OTel spans
// following the GenAI semantic conventions. Each operation produces a span
// named "embedding.<op>" carrying a gen_ai.operation.name attribute. Errors
// are recorded on the span and the status is set to StatusError on failure.
//
// Enable tracing by composing with other middleware:
//
//	emb = embedding.ApplyMiddleware(emb, embedding.WithTracing(), embedding.WithHooks(h))
func WithTracing() Middleware {
	return func(next Embedder) Embedder {
		return &tracedEmbedder{next: next}
	}
}

// tracedEmbedder wraps an Embedder and emits a span around each operation.
type tracedEmbedder struct {
	next Embedder
}

func (e *tracedEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	ctx, span := o11y.StartSpan(ctx, "embedding.embed", o11y.Attrs{
		o11y.AttrOperationName:  "embedding.embed",
		"embedding.input_count": len(texts),
	})
	defer span.End()

	vecs, err := e.next.Embed(ctx, texts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetAttributes(o11y.Attrs{"embedding.result_count": len(vecs)})
	span.SetStatus(o11y.StatusOK, "")
	return vecs, nil
}

func (e *tracedEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	ctx, span := o11y.StartSpan(ctx, "embedding.embed_single", o11y.Attrs{
		o11y.AttrOperationName: "embedding.embed_single",
	})
	defer span.End()

	vec, err := e.next.EmbedSingle(ctx, text)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetStatus(o11y.StatusOK, "")
	return vec, nil
}

func (e *tracedEmbedder) Dimensions() int {
	return e.next.Dimensions()
}

// Ensure tracedEmbedder implements Embedder at compile time.
var _ Embedder = (*tracedEmbedder)(nil)
