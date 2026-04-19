package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/o11y"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestExecutor is a minimal DurableExecutor used to drive the tracing
// middleware in tests. Errors and results on each method can be configured.
type tracingTestExecutor struct {
	execErr    error
	execHandle WorkflowHandle
	signalErr  error
	queryErr   error
	queryOut   any
	cancelErr  error
}

func (e *tracingTestExecutor) Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error) {
	return e.execHandle, e.execErr
}

func (e *tracingTestExecutor) Signal(ctx context.Context, workflowID string, signal Signal) error {
	return e.signalErr
}

func (e *tracingTestExecutor) Query(ctx context.Context, workflowID string, queryType string) (any, error) {
	return e.queryOut, e.queryErr
}

func (e *tracingTestExecutor) Cancel(ctx context.Context, workflowID string) error {
	return e.cancelErr
}

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("workflow-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForEveryOperation(t *testing.T) {
	exporter := setupTracing(t)

	base := &tracingTestExecutor{
		execHandle: &defaultHandle{id: "wf-1", runID: "run-1", status: StatusCompleted, done: make(chan struct{})},
		queryOut:   "status",
	}
	exec := ApplyMiddleware(DurableExecutor(base), WithTracing())

	ctx := context.Background()

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name: "execute",
			run: func() error {
				_, err := exec.Execute(ctx, func(WorkflowContext, any) (any, error) { return nil, nil }, WorkflowOptions{ID: "wf-1"})
				return err
			},
			spanOp: "workflow.execute",
		},
		{
			name:   "signal",
			run:    func() error { return exec.Signal(ctx, "wf-1", Signal{Name: "s1"}) },
			spanOp: "workflow.signal",
		},
		{
			name: "query",
			run: func() error {
				_, err := exec.Query(ctx, "wf-1", "status")
				return err
			},
			spanOp: "workflow.query",
		},
		{
			name:   "cancel",
			run:    func() error { return exec.Cancel(ctx, "wf-1") },
			spanOp: "workflow.cancel",
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
	base := &tracingTestExecutor{signalErr: wantErr}
	exec := ApplyMiddleware(DurableExecutor(base), WithTracing())

	err := exec.Signal(context.Background(), "wf-1", Signal{Name: "s1"})
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
