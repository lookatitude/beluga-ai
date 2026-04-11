package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/schema"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestMemory is a minimal Memory used to drive the tracing middleware
// in tests. Errors on each method can be configured independently.
type tracingTestMemory struct {
	saveErr   error
	loadErr   error
	loadMsgs  []schema.Message
	searchErr error
	searchDoc []schema.Document
	clearErr  error
}

func (m *tracingTestMemory) Save(ctx context.Context, input, output schema.Message) error {
	return m.saveErr
}

func (m *tracingTestMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	return m.loadMsgs, m.loadErr
}

func (m *tracingTestMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return m.searchDoc, m.searchErr
}

func (m *tracingTestMemory) Clear(ctx context.Context) error {
	return m.clearErr
}

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("memory-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForEveryOperation(t *testing.T) {
	exporter := setupTracing(t)

	base := &tracingTestMemory{
		loadMsgs:  []schema.Message{schema.NewHumanMessage("hi")},
		searchDoc: []schema.Document{{ID: "1", Content: "x"}, {ID: "2", Content: "y"}},
	}
	mem := ApplyMiddleware(Memory(base), WithTracing())

	ctx := context.Background()

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name:   "save",
			run:    func() error { return mem.Save(ctx, schema.NewHumanMessage("in"), schema.NewAIMessage("out")) },
			spanOp: "memory.save",
		},
		{
			name: "load",
			run: func() error {
				_, err := mem.Load(ctx, "q")
				return err
			},
			spanOp: "memory.load",
		},
		{
			name: "search",
			run: func() error {
				_, err := mem.Search(ctx, "q", 5)
				return err
			},
			spanOp: "memory.search",
		},
		{
			name:   "clear",
			run:    func() error { return mem.Clear(ctx) },
			spanOp: "memory.clear",
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
	base := &tracingTestMemory{saveErr: wantErr}
	mem := ApplyMiddleware(Memory(base), WithTracing())

	err := mem.Save(context.Background(), schema.NewHumanMessage("in"), schema.NewAIMessage("out"))
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
