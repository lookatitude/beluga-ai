package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/schema"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestRouter is a minimal CostRouter used to drive the tracing
// middleware in tests.
type tracingTestRouter struct {
	result ModelSelection
	err    error
}

func (r *tracingTestRouter) SelectModel(ctx context.Context, msgs []schema.Message) (ModelSelection, error) {
	return r.result, r.err
}

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("routing-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpanForSelectModel(t *testing.T) {
	exporter := setupTracing(t)

	base := &tracingTestRouter{
		result: ModelSelection{ModelID: "gpt-4", Tier: TierLarge, EstimatedCost: 0.1, Reason: "test"},
	}
	router := ApplyMiddleware(CostRouter(base), WithTracing())

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name: "select_model",
			run: func() error {
				_, err := router.SelectModel(context.Background(), []schema.Message{schema.NewHumanMessage("hi")})
				return err
			},
			spanOp: "llm.routing.select_model",
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

	wantErr := core.Errorf(core.ErrProviderDown, "router down")
	base := &tracingTestRouter{err: wantErr}
	router := ApplyMiddleware(CostRouter(base), WithTracing())

	_, err := router.SelectModel(context.Background(), []schema.Message{schema.NewHumanMessage("hi")})
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
