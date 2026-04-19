package retriever

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/o11y"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestRetriever is a minimal Retriever used to drive the tracing
// middleware in tests.
type tracingTestRetriever struct {
	docs []schema.Document
	err  error
}

func (r *tracingTestRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	return r.docs, r.err
}

func setupRetrieverTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("retriever-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestRetrieverWithTracing_EmitsSpan(t *testing.T) {
	exporter := setupRetrieverTracing(t)

	base := &tracingTestRetriever{
		docs: []schema.Document{{ID: "1", Content: "x"}, {ID: "2", Content: "y"}},
	}
	r := ApplyMiddleware(Retriever(base), WithTracing())

	docs, err := r.Retrieve(context.Background(), "q")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	const wantName = "rag.retriever.retrieve"
	if spans[0].Name != wantName {
		t.Errorf("expected span name %q, got %q", wantName, spans[0].Name)
	}

	var opAttrFound bool
	for _, attr := range spans[0].Attributes {
		if string(attr.Key) == o11y.AttrOperationName && attr.Value.AsString() == wantName {
			opAttrFound = true
			break
		}
	}
	if !opAttrFound {
		t.Errorf("expected %s=%q attribute on span", o11y.AttrOperationName, wantName)
	}
}

func TestRetrieverWithTracing_RecordsErrorOnFailure(t *testing.T) {
	exporter := setupRetrieverTracing(t)

	wantErr := core.Errorf(core.ErrProviderDown, "backend down")
	base := &tracingTestRetriever{err: wantErr}
	r := ApplyMiddleware(Retriever(base), WithTracing())

	_, err := r.Retrieve(context.Background(), "q")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error to wrap %v, got %v", wantErr, err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if len(spans[0].Events) == 0 {
		t.Errorf("expected RecordError to add an event to the span, got none")
	}
	if spans[0].Status.Code.String() != "Error" {
		t.Errorf("expected span status Error, got %v", spans[0].Status.Code)
	}
}
