package retriever

import (
	"context"

	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/schema"
)

// WithTracing returns middleware that wraps a Retriever with OTel spans
// following the GenAI semantic conventions. Each Retrieve call produces a
// span named "rag.retriever.retrieve" carrying a gen_ai.operation.name
// attribute. Errors are recorded on the span and the status is set to
// StatusError on failure.
//
// Enable tracing by composing with other middleware:
//
//	r = retriever.ApplyMiddleware(r, retriever.WithTracing(), retriever.WithHooks(h))
func WithTracing() Middleware {
	return func(next Retriever) Retriever {
		return &tracedRetriever{next: next}
	}
}

// tracedRetriever wraps a Retriever and emits a span around each operation.
type tracedRetriever struct {
	next Retriever
}

func (r *tracedRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	ctx, span := o11y.StartSpan(ctx, "rag.retriever.retrieve", o11y.Attrs{
		o11y.AttrOperationName: "rag.retriever.retrieve",
	})
	defer span.End()

	docs, err := r.next.Retrieve(ctx, query, opts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetAttributes(o11y.Attrs{"rag.retriever.result_count": len(docs)})
	span.SetStatus(o11y.StatusOK, "")
	return docs, nil
}

// Ensure tracedRetriever implements Retriever at compile time.
var _ Retriever = (*tracedRetriever)(nil)
