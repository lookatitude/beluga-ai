package splitter

import (
	"context"

	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/schema"
)

// WithTracing returns middleware that wraps a TextSplitter with OTel spans
// following the GenAI semantic conventions. Each operation produces a span
// named "rag.splitter.<op>" carrying a gen_ai.operation.name attribute.
// Errors are recorded on the span and the status is set to StatusError on
// failure.
//
// Enable tracing by composing with other middleware:
//
//	s = splitter.ApplyMiddleware(s, splitter.WithTracing())
func WithTracing() Middleware {
	return func(next TextSplitter) TextSplitter {
		return &tracedSplitter{next: next}
	}
}

// tracedSplitter wraps a TextSplitter and emits a span around each operation.
type tracedSplitter struct {
	next TextSplitter
}

func (s *tracedSplitter) Split(ctx context.Context, text string) ([]string, error) {
	ctx, span := o11y.StartSpan(ctx, "rag.splitter.split", o11y.Attrs{
		o11y.AttrOperationName:            "rag.splitter.split",
		"rag.splitter.split.input_length": len(text),
	})
	defer span.End()

	chunks, err := s.next.Split(ctx, text)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetAttributes(o11y.Attrs{"rag.splitter.split.chunk_count": len(chunks)})
	span.SetStatus(o11y.StatusOK, "")
	return chunks, nil
}

func (s *tracedSplitter) SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error) {
	ctx, span := o11y.StartSpan(ctx, "rag.splitter.split_documents", o11y.Attrs{
		o11y.AttrOperationName:                   "rag.splitter.split_documents",
		"rag.splitter.split_documents.doc_count": len(docs),
	})
	defer span.End()

	out, err := s.next.SplitDocuments(ctx, docs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetAttributes(o11y.Attrs{"rag.splitter.split_documents.result_count": len(out)})
	span.SetStatus(o11y.StatusOK, "")
	return out, nil
}

// Ensure tracedSplitter implements TextSplitter at compile time.
var _ TextSplitter = (*tracedSplitter)(nil)
