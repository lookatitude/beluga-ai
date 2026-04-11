package server

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestAdapter is a minimal ServerAdapter used to drive the tracing
// middleware in tests. Errors on each method can be configured independently.
type tracingTestAdapter struct {
	registerAgentErr   error
	registerHandlerErr error
	serveErr           error
	shutdownErr        error
}

func (t *tracingTestAdapter) RegisterAgent(path string, a agent.Agent) error {
	return t.registerAgentErr
}

func (t *tracingTestAdapter) RegisterHandler(path string, handler http.Handler) error {
	return t.registerHandlerErr
}

func (t *tracingTestAdapter) Serve(ctx context.Context, addr string) error {
	return t.serveErr
}

func (t *tracingTestAdapter) Shutdown(ctx context.Context) error {
	return t.shutdownErr
}

func setupServerTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("server-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForEveryOperation(t *testing.T) {
	exporter := setupServerTracing(t)

	base := &tracingTestAdapter{}
	s := ApplyMiddleware(ServerAdapter(base), WithTracing())

	ctx := context.Background()
	ag := &mockAgent{id: "test", result: "ok"}

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name:   "register_agent",
			run:    func() error { return s.RegisterAgent("/x", ag) },
			spanOp: "server.register_agent",
		},
		{
			name: "register_handler",
			run: func() error {
				return s.RegisterHandler("/x", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			},
			spanOp: "server.register_handler",
		},
		{
			name:   "serve",
			run:    func() error { return s.Serve(ctx, ":0") },
			spanOp: "server.serve",
		},
		{
			name:   "shutdown",
			run:    func() error { return s.Shutdown(ctx) },
			spanOp: "server.shutdown",
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
	exporter := setupServerTracing(t)

	wantErr := core.Errorf(core.ErrProviderDown, "adapter down")
	base := &tracingTestAdapter{serveErr: wantErr}
	s := ApplyMiddleware(ServerAdapter(base), WithTracing())

	err := s.Serve(context.Background(), ":0")
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
