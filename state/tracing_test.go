package state

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/o11y"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// tracingTestStore is a minimal Store used to drive the tracing middleware
// in tests. Errors on each method can be configured independently.
type tracingTestStore struct {
	getErr    error
	setErr    error
	deleteErr error
	watchErr  error
	closeErr  error
}

func (s *tracingTestStore) Get(ctx context.Context, key string) (any, error) {
	return nil, s.getErr
}

func (s *tracingTestStore) Set(ctx context.Context, key string, value any) error {
	return s.setErr
}

func (s *tracingTestStore) Delete(ctx context.Context, key string) error {
	return s.deleteErr
}

func (s *tracingTestStore) Watch(ctx context.Context, key string) (<-chan StateChange, error) {
	if s.watchErr != nil {
		return nil, s.watchErr
	}
	ch := make(chan StateChange, 1)
	close(ch)
	return ch, nil
}

func (s *tracingTestStore) Close() error {
	return s.closeErr
}

func setupStateTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("state-test", o11y.WithSpanExporter(exporter), o11y.WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	t.Cleanup(shutdown)
	return exporter
}

func TestWithTracing_EmitsSpansForEveryOperation(t *testing.T) {
	exporter := setupStateTracing(t)

	base := &tracingTestStore{}
	s := ApplyMiddleware(Store(base), WithTracing())

	ctx := context.Background()

	cases := []struct {
		name   string
		run    func() error
		spanOp string
	}{
		{
			name: "get",
			run: func() error {
				_, err := s.Get(ctx, "k")
				return err
			},
			spanOp: "state.get",
		},
		{
			name:   "set",
			run:    func() error { return s.Set(ctx, "k", "v") },
			spanOp: "state.set",
		},
		{
			name:   "delete",
			run:    func() error { return s.Delete(ctx, "k") },
			spanOp: "state.delete",
		},
		{
			name: "watch",
			run: func() error {
				_, err := s.Watch(ctx, "k")
				return err
			},
			spanOp: "state.watch",
		},
		{
			name:   "close",
			run:    func() error { return s.Close() },
			spanOp: "state.close",
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
	exporter := setupStateTracing(t)

	wantErr := core.Errorf(core.ErrProviderDown, "backend down")
	base := &tracingTestStore{setErr: wantErr}
	s := ApplyMiddleware(Store(base), WithTracing())

	err := s.Set(context.Background(), "k", "v")
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
