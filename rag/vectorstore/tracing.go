package vectorstore

import (
	"context"

	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/schema"
)

// WithTracing returns middleware that wraps a VectorStore with OTel spans
// following the GenAI semantic conventions. Each operation produces a span
// named "rag.vectorstore.<op>" carrying a gen_ai.operation.name attribute.
// Errors are recorded on the span and the status is set to StatusError on
// failure.
//
// Enable tracing by composing with other middleware:
//
//	store = vectorstore.ApplyMiddleware(store, vectorstore.WithTracing(), vectorstore.WithHooks(h))
func WithTracing() Middleware {
	return func(next VectorStore) VectorStore {
		return &tracedStore{next: next}
	}
}

// tracedStore wraps a VectorStore and emits a span around each operation.
type tracedStore struct {
	next VectorStore
}

func (s *tracedStore) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	ctx, span := o11y.StartSpan(ctx, "rag.vectorstore.add", o11y.Attrs{
		o11y.AttrOperationName:      "rag.vectorstore.add",
		"rag.vectorstore.add.count": len(docs),
	})
	defer span.End()

	err := s.next.Add(ctx, docs, embeddings)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

func (s *tracedStore) Search(ctx context.Context, query []float32, k int, opts ...SearchOption) ([]schema.Document, error) {
	ctx, span := o11y.StartSpan(ctx, "rag.vectorstore.search", o11y.Attrs{
		o11y.AttrOperationName:     "rag.vectorstore.search",
		"rag.vectorstore.search.k": k,
	})
	defer span.End()

	results, err := s.next.Search(ctx, query, k, opts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetAttributes(o11y.Attrs{"rag.vectorstore.search.result_count": len(results)})
	span.SetStatus(o11y.StatusOK, "")
	return results, nil
}

func (s *tracedStore) Delete(ctx context.Context, ids []string) error {
	ctx, span := o11y.StartSpan(ctx, "rag.vectorstore.delete", o11y.Attrs{
		o11y.AttrOperationName:         "rag.vectorstore.delete",
		"rag.vectorstore.delete.count": len(ids),
	})
	defer span.End()

	err := s.next.Delete(ctx, ids)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

// Ensure tracedStore implements VectorStore at compile time.
var _ VectorStore = (*tracedStore)(nil)
