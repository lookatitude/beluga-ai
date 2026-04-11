package embedding

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestEmbedder is a minimal Embedder used to drive the tracing
// middleware in tests. Errors and return values for each method can be
// configured independently.
type tracingTestEmbedder struct {
	embedErr   error
	embedVecs  [][]float32
	singleErr  error
	singleVec  []float32
	dimensions int
}

func (e *tracingTestEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return e.embedVecs, e.embedErr
}

func (e *tracingTestEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	return e.singleVec, e.singleErr
}

func (e *tracingTestEmbedder) Dimensions() int { return e.dimensions }

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("embedding-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForEveryOperation(t *testing.T) {
	exporter := setupTracing(t)

	base := &tracingTestEmbedder{
		embedVecs:  [][]float32{{1, 2, 3}, {4, 5, 6}},
		singleVec:  []float32{1, 2, 3},
		dimensions: 3,
	}
	emb := ApplyMiddleware(Embedder(base), WithTracing())

	ctx := context.Background()

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name: "embed",
			run: func() error {
				_, err := emb.Embed(ctx, []string{"a", "b"})
				return err
			},
			spanOp: "embedding.embed",
		},
		{
			name: "embed_single",
			run: func() error {
				_, err := emb.EmbedSingle(ctx, "a")
				return err
			},
			spanOp: "embedding.embed_single",
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
	base := &tracingTestEmbedder{embedErr: wantErr}
	emb := ApplyMiddleware(Embedder(base), WithTracing())

	_, err := emb.Embed(context.Background(), []string{"a"})
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

func TestWithTracing_Dimensions_Passthrough(t *testing.T) {
	_ = setupTracing(t)
	base := &tracingTestEmbedder{dimensions: 768}
	emb := ApplyMiddleware(Embedder(base), WithTracing())
	if got := emb.Dimensions(); got != 768 {
		t.Errorf("expected Dimensions() = 768, got %d", got)
	}
}
