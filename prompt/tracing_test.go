package prompt

import (
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/schema"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestManager is a minimal PromptManager used to drive the tracing
// middleware in tests. Errors on each method can be configured independently.
type tracingTestManager struct {
	getTmpl    *Template
	getErr     error
	renderMsgs []schema.Message
	renderErr  error
	listInfos  []TemplateInfo
}

func (m *tracingTestManager) Get(name, version string) (*Template, error) {
	return m.getTmpl, m.getErr
}

func (m *tracingTestManager) Render(name string, vars map[string]any) ([]schema.Message, error) {
	return m.renderMsgs, m.renderErr
}

func (m *tracingTestManager) List() []TemplateInfo {
	return m.listInfos
}

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("prompt-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForEveryOperation(t *testing.T) {
	exporter := setupTracing(t)

	base := &tracingTestManager{
		getTmpl:    &Template{Name: "t", Version: "1", Content: "hi"},
		renderMsgs: []schema.Message{schema.NewSystemMessage("hi")},
		listInfos:  []TemplateInfo{{Name: "t", Version: "1"}},
	}
	mgr := ApplyMiddleware(PromptManager(base), WithTracing())

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name: "get",
			run: func() error {
				_, err := mgr.Get("t", "1")
				return err
			},
			spanOp: "prompt.get",
		},
		{
			name: "render",
			run: func() error {
				_, err := mgr.Render("t", nil)
				return err
			},
			spanOp: "prompt.render",
		},
		{
			name: "list",
			run: func() error {
				_ = mgr.List()
				return nil
			},
			spanOp: "prompt.list",
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
	base := &tracingTestManager{getErr: wantErr}
	mgr := ApplyMiddleware(PromptManager(base), WithTracing())

	_, err := mgr.Get("t", "1")
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
