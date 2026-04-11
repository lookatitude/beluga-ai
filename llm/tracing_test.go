package llm

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/schema"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestModel is a minimal ChatModel used to drive the tracing
// middleware in tests.
type tracingTestModel struct {
	modelID     string
	generateErr error
	generateOut *schema.AIMessage
	streamErr   error
	streamOut   []schema.StreamChunk
}

func (m *tracingTestModel) Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	return m.generateOut, nil
}

func (m *tracingTestModel) Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {
		if m.streamErr != nil {
			yield(schema.StreamChunk{}, m.streamErr)
			return
		}
		for _, c := range m.streamOut {
			if !yield(c, nil) {
				return
			}
		}
	}
}

func (m *tracingTestModel) BindTools(tools []schema.ToolDefinition) ChatModel { return m }
func (m *tracingTestModel) ModelID() string                                   { return m.modelID }

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("llm-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForEveryOperation(t *testing.T) {
	exporter := setupTracing(t)

	base := &tracingTestModel{
		modelID:     "test-model",
		generateOut: &schema.AIMessage{ModelID: "test-model", Usage: schema.Usage{InputTokens: 3, OutputTokens: 5}},
		streamOut: []schema.StreamChunk{
			{Delta: "hello"},
			{Delta: " world"},
		},
	}
	model := ApplyMiddleware(ChatModel(base), WithTracing())

	ctx := context.Background()

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name: "generate",
			run: func() error {
				_, err := model.Generate(ctx, []schema.Message{schema.NewHumanMessage("hi")})
				return err
			},
			spanOp: "llm.generate",
		},
		{
			name: "stream",
			run: func() error {
				for _, err := range model.Stream(ctx, []schema.Message{schema.NewHumanMessage("hi")}) {
					if err != nil {
						return err
					}
				}
				return nil
			},
			spanOp: "llm.stream",
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
	base := &tracingTestModel{modelID: "test-model", generateErr: wantErr}
	model := ApplyMiddleware(ChatModel(base), WithTracing())

	_, err := model.Generate(context.Background(), []schema.Message{schema.NewHumanMessage("hi")})
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
