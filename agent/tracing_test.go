package agent

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/tool"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestAgent is a minimal Agent used to drive the tracing middleware in
// tests. Errors on each method can be configured independently.
type tracingTestAgent struct {
	id           string
	invokeResult string
	invokeErr    error
	streamEvents []Event
	streamErr    error
}

func (a *tracingTestAgent) ID() string         { return a.id }
func (a *tracingTestAgent) Persona() Persona   { return Persona{} }
func (a *tracingTestAgent) Tools() []tool.Tool { return nil }
func (a *tracingTestAgent) Children() []Agent  { return nil }
func (a *tracingTestAgent) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	return a.invokeResult, a.invokeErr
}
func (a *tracingTestAgent) Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		for _, e := range a.streamEvents {
			if !yield(e, nil) {
				return
			}
		}
		if a.streamErr != nil {
			yield(Event{}, a.streamErr)
		}
	}
}

func setupAgentTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("agent-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpanForInvoke(t *testing.T) {
	exporter := setupAgentTracing(t)

	base := &tracingTestAgent{id: "alice", invokeResult: "hello"}
	a := ApplyMiddleware(Agent(base), WithTracing())

	out, err := a.Invoke(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}
	if out != "hello" {
		t.Errorf("Invoke: got %q, want %q", out, "hello")
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "agent.invoke" {
		t.Errorf("expected span name %q, got %q", "agent.invoke", spans[0].Name)
	}

	var opFound, nameFound bool
	for _, attr := range spans[0].Attributes {
		switch string(attr.Key) {
		case o11y.AttrOperationName:
			if attr.Value.AsString() == "agent.invoke" {
				opFound = true
			}
		case o11y.AttrAgentName:
			if attr.Value.AsString() == "alice" {
				nameFound = true
			}
		}
	}
	if !opFound {
		t.Errorf("expected %s=agent.invoke attribute on span", o11y.AttrOperationName)
	}
	if !nameFound {
		t.Errorf("expected %s=alice attribute on span", o11y.AttrAgentName)
	}
}

func TestWithTracing_RecordsErrorOnInvokeFailure(t *testing.T) {
	exporter := setupAgentTracing(t)

	wantErr := core.Errorf(core.ErrProviderDown, "backend down")
	base := &tracingTestAgent{id: "bob", invokeErr: wantErr}
	a := ApplyMiddleware(Agent(base), WithTracing())

	_, err := a.Invoke(context.Background(), "hi")
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

func TestWithTracing_EmitsSpanForStream(t *testing.T) {
	exporter := setupAgentTracing(t)

	base := &tracingTestAgent{
		id: "carol",
		streamEvents: []Event{
			{Type: EventText, Text: "hi"},
			{Type: EventDone},
		},
	}
	a := ApplyMiddleware(Agent(base), WithTracing())

	var count int
	for event, err := range a.Stream(context.Background(), "hi") {
		if err != nil {
			t.Fatalf("stream: unexpected error: %v", err)
		}
		_ = event
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 events, got %d", count)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "agent.stream" {
		t.Errorf("expected span name %q, got %q", "agent.stream", spans[0].Name)
	}
}

func TestWithTracing_RecordsErrorOnStreamFailure(t *testing.T) {
	exporter := setupAgentTracing(t)

	wantErr := core.Errorf(core.ErrProviderDown, "stream broke")
	base := &tracingTestAgent{id: "dave", streamErr: wantErr}
	a := ApplyMiddleware(Agent(base), WithTracing())

	var gotErr error
	for _, err := range a.Stream(context.Background(), "hi") {
		if err != nil {
			gotErr = err
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
