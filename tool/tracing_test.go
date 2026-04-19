package tool

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/o11y"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestTool is a minimal Tool used to drive the tracing middleware in
// tests. Errors on Execute can be configured.
type tracingTestTool struct {
	name       string
	execErr    error
	execResult *Result
}

func (t *tracingTestTool) Name() string                { return t.name }
func (t *tracingTestTool) Description() string         { return "test tool" }
func (t *tracingTestTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *tracingTestTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
	if t.execErr != nil {
		return nil, t.execErr
	}
	return t.execResult, nil
}

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("tool-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForEveryOperation(t *testing.T) {
	exporter := setupTracing(t)

	base := &tracingTestTool{
		name:       "echo",
		execResult: TextResult("ok"),
	}
	tool := ApplyMiddleware(Tool(base), WithTracing())

	ctx := context.Background()

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name: "execute",
			run: func() error {
				_, err := tool.Execute(ctx, map[string]any{"q": "v"})
				return err
			},
			spanOp: "tool.execute",
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

	wantErr := core.Errorf(core.ErrToolFailed, "boom")
	base := &tracingTestTool{name: "echo", execErr: wantErr}
	tool := ApplyMiddleware(Tool(base), WithTracing())

	_, err := tool.Execute(context.Background(), map[string]any{})
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

// TestWithTracing_PassthroughMethods verifies that tracedTool forwards the
// non-spanned metadata methods (Name, Description, InputSchema) unchanged.
func TestWithTracing_PassthroughMethods(t *testing.T) {
	base := &tracingTestTool{name: "calc"}
	wrapped := ApplyMiddleware(Tool(base), WithTracing())

	if got := wrapped.Name(); got != "calc" {
		t.Errorf("Name() = %q, want %q", got, "calc")
	}
	if got := wrapped.Description(); got != "test tool" {
		t.Errorf("Description() = %q, want %q", got, "test tool")
	}
	if got := wrapped.InputSchema(); got == nil {
		t.Error("InputSchema() = nil, want non-nil")
	}
}
