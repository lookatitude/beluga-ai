package vectorstore

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/schema"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestStore is a minimal VectorStore used to drive the tracing
// middleware in tests. Errors on each method can be configured independently.
type tracingTestStore struct {
	addErr     error
	searchErr  error
	searchDocs []schema.Document
	deleteErr  error
}

func (s *tracingTestStore) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	return s.addErr
}

func (s *tracingTestStore) Search(ctx context.Context, query []float32, k int, opts ...SearchOption) ([]schema.Document, error) {
	return s.searchDocs, s.searchErr
}

func (s *tracingTestStore) Delete(ctx context.Context, ids []string) error {
	return s.deleteErr
}

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("vectorstore-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForEveryOperation(t *testing.T) {
	exporter := setupTracing(t)

	base := &tracingTestStore{
		searchDocs: []schema.Document{{ID: "1", Content: "x"}, {ID: "2", Content: "y"}},
	}
	store := ApplyMiddleware(VectorStore(base), WithTracing())

	ctx := context.Background()

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name: "add",
			run: func() error {
				return store.Add(ctx, []schema.Document{{ID: "1", Content: "hi"}}, [][]float32{{0.1, 0.2}})
			},
			spanOp: "rag.vectorstore.add",
		},
		{
			name: "search",
			run: func() error {
				_, err := store.Search(ctx, []float32{0.1, 0.2}, 5)
				return err
			},
			spanOp: "rag.vectorstore.search",
		},
		{
			name:   "delete",
			run:    func() error { return store.Delete(ctx, []string{"1"}) },
			spanOp: "rag.vectorstore.delete",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exporter.Reset()
			if err := tc.run(); err != nil {
				t.Fatalf("%s: unexpected error: %v", tc.name, err)
			}

			spans := exporter.GetSpans()
			if len(spans) != 1 {
				t.Fatalf("%s: expected 1 span, got %d", tc.name, len(spans))
			}
			if spans[0].Name != tc.spanOp {
				t.Errorf("%s: expected span name %q, got %q", tc.name, tc.spanOp, spans[0].Name)
			}

			var opAttrFound bool
			for _, attr := range spans[0].Attributes {
				if string(attr.Key) == o11y.AttrOperationName && attr.Value.AsString() == tc.spanOp {
					opAttrFound = true
					break
				}
			}
			if !opAttrFound {
				t.Errorf("%s: expected %s=%q attribute on span", tc.name, o11y.AttrOperationName, tc.spanOp)
			}
		})
	}
}

func TestWithTracing_RecordsErrorOnFailure(t *testing.T) {
	exporter := setupTracing(t)

	wantErr := core.Errorf(core.ErrProviderDown, "backend down")
	base := &tracingTestStore{addErr: wantErr}
	store := ApplyMiddleware(VectorStore(base), WithTracing())

	err := store.Add(context.Background(), []schema.Document{{ID: "1", Content: "hi"}}, [][]float32{{0.1}})
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
