package orchestration

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestRunnable is a minimal core.Runnable used to drive the tracing
// middleware in tests.
type tracingTestRunnable struct {
	invokeResult any
	invokeErr    error
	streamItems  []any
	streamErr    error
}

func (r *tracingTestRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	return r.invokeResult, r.invokeErr
}

func (r *tracingTestRunnable) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		for _, it := range r.streamItems {
			if !yield(it, nil) {
				return
			}
		}
		if r.streamErr != nil {
			yield(nil, r.streamErr)
		}
	}
}

func setupTracingOrch(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("orchestration-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForInvokeAndStream(t *testing.T) {
	exporter := setupTracingOrch(t)

	base := &tracingTestRunnable{
		invokeResult: "ok",
		streamItems:  []any{"a", "b"},
	}
	wrapped := ApplyMiddleware(core.Runnable(base), WithTracing())

	ctx := context.Background()

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name: "invoke",
			run: func() error {
				_, err := wrapped.Invoke(ctx, "in")
				return err
			},
			spanOp: "orchestration.invoke",
		},
		{
			name: "stream",
			run: func() error {
				for _, err := range wrapped.Stream(ctx, "in") {
					if err != nil {
						return err
					}
				}
				return nil
			},
			spanOp: "orchestration.stream",
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

func TestWithTracing_RecordsErrorOnInvokeFailure(t *testing.T) {
	exporter := setupTracingOrch(t)

	wantErr := core.Errorf(core.ErrProviderDown, "runnable down")
	base := &tracingTestRunnable{invokeErr: wantErr}
	wrapped := ApplyMiddleware(core.Runnable(base), WithTracing())

	_, err := wrapped.Invoke(context.Background(), "in")
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

func TestWithTracing_RecordsErrorOnStreamFailure(t *testing.T) {
	exporter := setupTracingOrch(t)

	wantErr := core.Errorf(core.ErrProviderDown, "stream down")
	base := &tracingTestRunnable{streamErr: wantErr}
	wrapped := ApplyMiddleware(core.Runnable(base), WithTracing())

	var gotErr error
	for _, err := range wrapped.Stream(context.Background(), "in") {
		if err != nil {
			gotErr = err
			break
		}
	}
	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("expected error to wrap %v, got %v", wantErr, gotErr)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Status.Code.String() != "Error" {
		t.Errorf("expected span status Error, got %v", spans[0].Status.Code)
	}
}
