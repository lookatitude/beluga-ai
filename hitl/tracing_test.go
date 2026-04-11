package hitl

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestManager is a minimal Manager used to drive the tracing middleware
// in tests. Errors and return values on each method can be configured
// independently.
type tracingTestManager struct {
	requestResp   *InteractionResponse
	requestErr    error
	addPolicyErr  error
	shouldApprove bool
	shouldErr     error
	respondErr    error
}

func (m *tracingTestManager) RequestInteraction(ctx context.Context, req InteractionRequest) (*InteractionResponse, error) {
	return m.requestResp, m.requestErr
}

func (m *tracingTestManager) AddPolicy(policy ApprovalPolicy) error {
	return m.addPolicyErr
}

func (m *tracingTestManager) ShouldApprove(ctx context.Context, toolName string, confidence float64, risk RiskLevel) (bool, error) {
	return m.shouldApprove, m.shouldErr
}

func (m *tracingTestManager) Respond(ctx context.Context, requestID string, resp InteractionResponse) error {
	return m.respondErr
}

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("hitl-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForEveryOperation(t *testing.T) {
	exporter := setupTracing(t)

	base := &tracingTestManager{
		requestResp:   &InteractionResponse{Decision: DecisionApprove},
		shouldApprove: true,
	}
	mgr := ApplyMiddleware(Manager(base), WithTracing())

	ctx := context.Background()

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name: "request_interaction",
			run: func() error {
				_, err := mgr.RequestInteraction(ctx, InteractionRequest{ID: "r1", Type: TypeApproval, ToolName: "tool"})
				return err
			},
			spanOp: "hitl.request_interaction",
		},
		{
			name:   "add_policy",
			run:    func() error { return mgr.AddPolicy(ApprovalPolicy{Name: "p1", ToolPattern: "*"}) },
			spanOp: "hitl.add_policy",
		},
		{
			name: "should_approve",
			run: func() error {
				_, err := mgr.ShouldApprove(ctx, "tool", 0.9, RiskReadOnly)
				return err
			},
			spanOp: "hitl.should_approve",
		},
		{
			name:   "respond",
			run:    func() error { return mgr.Respond(ctx, "r1", InteractionResponse{Decision: DecisionApprove}) },
			spanOp: "hitl.respond",
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
	base := &tracingTestManager{requestErr: wantErr}
	mgr := ApplyMiddleware(Manager(base), WithTracing())

	_, err := mgr.RequestInteraction(context.Background(), InteractionRequest{ID: "r1"})
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
