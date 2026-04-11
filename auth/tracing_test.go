package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestPolicy is a minimal Policy used to drive the tracing middleware
// in tests. Decisions and errors can be configured independently.
type tracingTestPolicy struct {
	name    string
	allowed bool
	err     error
}

func (p *tracingTestPolicy) Name() string { return p.name }

func (p *tracingTestPolicy) Authorize(ctx context.Context, subject string, permission Permission, resource string) (bool, error) {
	return p.allowed, p.err
}

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("auth-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpanForAuthorize(t *testing.T) {
	cases := []struct {
		name     string
		allowed  bool
		decision string
	}{
		{name: "allow", allowed: true, decision: "allow"},
		{name: "deny", allowed: false, decision: "deny"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exporter := setupTracing(t)
			runAuthorizeAndAssert(t, exporter, tc.allowed, tc.decision)
		})
	}
}

func runAuthorizeAndAssert(t *testing.T, exporter *tracetest.InMemoryExporter, allowedIn bool, decision string) {
	t.Helper()
	base := &tracingTestPolicy{name: "test", allowed: allowedIn}
	pol := ApplyMiddleware(Policy(base), WithTracing())

	allowed, err := pol.Authorize(context.Background(), "alice", PermToolExec, "calc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed != allowedIn {
		t.Fatalf("expected allowed=%v, got %v", allowedIn, allowed)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "auth.authorize" {
		t.Errorf("expected span name %q, got %q", "auth.authorize", spans[0].Name)
	}
	assertSpanAttr(t, spans[0].Attributes, o11y.AttrOperationName, "auth.authorize")
	assertSpanAttr(t, spans[0].Attributes, "auth.decision", decision)
}

func assertSpanAttr(t *testing.T, attrs []attribute.KeyValue, key, want string) {
	t.Helper()
	for _, attr := range attrs {
		if string(attr.Key) == key && attr.Value.AsString() == want {
			return
		}
	}
	t.Errorf("expected %s=%q attribute on span", key, want)
}

func TestWithTracing_RecordsErrorOnFailure(t *testing.T) {
	exporter := setupTracing(t)

	wantErr := core.Errorf(core.ErrProviderDown, "opa down")
	base := &tracingTestPolicy{name: "test", err: wantErr}
	pol := ApplyMiddleware(Policy(base), WithTracing())

	_, err := pol.Authorize(context.Background(), "alice", PermToolExec, "calc")
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

// TestWithTracing_NamePassthrough verifies tracedPolicy forwards Name().
func TestWithTracing_NamePassthrough(t *testing.T) {
	base := &tracingTestPolicy{name: "rbac-test", allowed: true}
	wrapped := ApplyMiddleware(Policy(base), WithTracing())
	if got := wrapped.Name(); got != "rbac-test" {
		t.Errorf("Name() = %q, want %q", got, "rbac-test")
	}
}
