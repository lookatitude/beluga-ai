package splitter

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/schema"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestSplitter is a minimal TextSplitter used to drive the tracing
// middleware in tests. Errors on each method can be configured independently.
type tracingTestSplitter struct {
	splitErr     error
	splitChunks  []string
	splitDocsErr error
	splitDocsOut []schema.Document
}

func (s *tracingTestSplitter) Split(ctx context.Context, text string) ([]string, error) {
	return s.splitChunks, s.splitErr
}

func (s *tracingTestSplitter) SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error) {
	return s.splitDocsOut, s.splitDocsErr
}

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("splitter-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForEveryOperation(t *testing.T) {
	exporter := setupTracing(t)

	base := &tracingTestSplitter{
		splitChunks:  []string{"a", "b"},
		splitDocsOut: []schema.Document{{ID: "1#chunk0", Content: "a"}, {ID: "1#chunk1", Content: "b"}},
	}
	sp := ApplyMiddleware(TextSplitter(base), WithTracing())

	ctx := context.Background()

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name: "split",
			run: func() error {
				_, err := sp.Split(ctx, "hello world")
				return err
			},
			spanOp: "rag.splitter.split",
		},
		{
			name: "split_documents",
			run: func() error {
				_, err := sp.SplitDocuments(ctx, []schema.Document{{ID: "1", Content: "hello"}})
				return err
			},
			spanOp: "rag.splitter.split_documents",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exporter.Reset()
			if err := tc.run(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertSingleTracingSpan(t, exporter, tc.spanOp)
		})
	}
}

// assertSingleTracingSpan asserts exactly one span was recorded with the
// expected name and operation attribute.
func assertSingleTracingSpan(t *testing.T, exporter *tracetest.InMemoryExporter, spanOp string) {
	t.Helper()
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != spanOp {
		t.Errorf("expected span name %q, got %q", spanOp, spans[0].Name)
	}
	for _, attr := range spans[0].Attributes {
		if string(attr.Key) == o11y.AttrOperationName && attr.Value.AsString() == spanOp {
			return
		}
	}
	t.Errorf("expected %s=%q attribute on span", o11y.AttrOperationName, spanOp)
}

func TestWithTracing_RecordsErrorOnFailure(t *testing.T) {
	exporter := setupTracing(t)

	wantErr := core.Errorf(core.ErrProviderDown, "backend down")
	base := &tracingTestSplitter{splitErr: wantErr}
	sp := ApplyMiddleware(TextSplitter(base), WithTracing())

	_, err := sp.Split(context.Background(), "hello")
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
