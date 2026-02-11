package o11y

import (
	"context"
	"errors"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestStartSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := InitTracer("test-service", WithSpanExporter(exporter), WithSyncExport())
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	defer shutdown()

	ctx := context.Background()

	t.Run("creates span with attributes", func(t *testing.T) {
		exporter.Reset()

		ctx, span := StartSpan(ctx, "test-op", Attrs{
			AttrAgentName:     "test-agent",
			AttrOperationName: "chat",
			AttrRequestModel:  "gpt-4o",
		})
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
		span.End()

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}
		if spans[0].Name != "test-op" {
			t.Errorf("expected span name 'test-op', got %q", spans[0].Name)
		}
	})

	t.Run("span SetAttributes adds attributes", func(t *testing.T) {
		exporter.Reset()

		_, span := StartSpan(ctx, "attr-test", nil)
		span.SetAttributes(Attrs{
			AttrResponseModel: "gpt-4o-mini",
			AttrInputTokens:   42,
		})
		span.End()

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}
		found := false
		for _, attr := range spans[0].Attributes {
			if string(attr.Key) == AttrResponseModel && attr.Value.AsString() == "gpt-4o-mini" {
				found = true
			}
		}
		if !found {
			t.Error("expected gen_ai.response.model attribute")
		}
	})

	t.Run("span RecordError records error event", func(t *testing.T) {
		exporter.Reset()

		_, span := StartSpan(ctx, "error-test", nil)
		span.RecordError(errors.New("test error"))
		span.End()

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}
		if len(spans[0].Events) == 0 {
			t.Error("expected at least one event for recorded error")
		}
	})

	t.Run("span SetStatus OK", func(t *testing.T) {
		exporter.Reset()

		_, span := StartSpan(ctx, "status-ok", nil)
		span.SetStatus(StatusOK, "all good")
		span.End()

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}
	})

	t.Run("span SetStatus Error", func(t *testing.T) {
		exporter.Reset()

		_, span := StartSpan(ctx, "status-err", nil)
		span.SetStatus(StatusError, "something failed")
		span.End()

		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}
	})

	t.Run("nil attrs does not panic", func(t *testing.T) {
		exporter.Reset()

		_, span := StartSpan(ctx, "nil-attrs", nil)
		span.SetAttributes(nil)
		span.End()
	})
}

func TestInitTracerWithSampler(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := InitTracer("sampled-service",
		WithSpanExporter(exporter),
		WithSampler(sdktrace.NeverSample()),
		WithSyncExport(),
	)
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	defer shutdown()

	_, span := StartSpan(context.Background(), "should-not-record", nil)
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 0 {
		t.Errorf("expected 0 spans with NeverSample, got %d", len(spans))
	}
}

func TestAttrsToOTel(t *testing.T) {
	tests := []struct {
		name  string
		attrs Attrs
		want  int
	}{
		{"nil attrs", nil, 0},
		{"empty attrs", Attrs{}, 0},
		{"string value", Attrs{"key": "val"}, 1},
		{"int value", Attrs{"key": 42}, 1},
		{"int64 value", Attrs{"key": int64(42)}, 1},
		{"float64 value", Attrs{"key": 3.14}, 1},
		{"bool value", Attrs{"key": true}, 1},
		{"unsupported type skipped", Attrs{"key": []int{1, 2}}, 0},
		{"mixed types", Attrs{"s": "v", "i": 1, "f": 1.0, "b": true}, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := attrsToOTel(tt.attrs)
			if len(got) != tt.want {
				t.Errorf("attrsToOTel(%v) returned %d attrs, want %d", tt.attrs, len(got), tt.want)
			}
		})
	}
}

func TestInitTracer_WithoutExporter(t *testing.T) {
	// InitTracer without exporter should succeed (no-op)
	shutdown, err := InitTracer("no-exporter-service")
	if err != nil {
		t.Fatalf("InitTracer without exporter: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown func")
	}
	shutdown()

	// Verify tracer works
	ctx, span := StartSpan(context.Background(), "test-span", nil)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	span.End()
}

func TestInitTracer_WithBatchedExport(t *testing.T) {
	// Test batched export (default when syncExport is false)
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := InitTracer("batched-service",
		WithSpanExporter(exporter),
		// Omit WithSyncExport() to test batched path
	)
	if err != nil {
		t.Fatalf("InitTracer with batched export: %v", err)
	}
	defer shutdown()

	_, span := StartSpan(context.Background(), "batched-span", nil)
	span.End()

	// Note: batched spans may not be available immediately,
	// but we're just testing that the code path doesn't error
}

func TestInitTracer_MultipleInits(t *testing.T) {
	// Test multiple InitTracer calls (should reinit)
	exporter1 := tracetest.NewInMemoryExporter()
	shutdown1, err := InitTracer("service-1", WithSpanExporter(exporter1), WithSyncExport())
	if err != nil {
		t.Fatalf("first InitTracer: %v", err)
	}
	shutdown1()

	// Second init with different service name
	exporter2 := tracetest.NewInMemoryExporter()
	shutdown2, err := InitTracer("service-2", WithSpanExporter(exporter2), WithSyncExport())
	if err != nil {
		t.Fatalf("second InitTracer: %v", err)
	}
	defer shutdown2()

	// Verify tracer works after reinit
	ctx, span := StartSpan(context.Background(), "after-reinit", Attrs{
		AttrAgentName: "test",
	})
	_ = ctx // ctx carries span for downstream propagation
	span.End()

	spans := exporter2.GetSpans()
	if len(spans) != 1 {
		t.Errorf("expected 1 span in second exporter, got %d", len(spans))
	}
}

func TestInitTracer_AllOptions(t *testing.T) {
	// Test all options together
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := InitTracer("full-options-service",
		WithSpanExporter(exporter),
		WithSampler(sdktrace.AlwaysSample()),
		WithSyncExport(),
	)
	if err != nil {
		t.Fatalf("InitTracer with all options: %v", err)
	}
	defer shutdown()

	ctx, span := StartSpan(context.Background(), "full-test", Attrs{
		AttrSystem:        "test-system",
		AttrOperationName: "test-op",
		AttrAgentName:     "test-agent",
	})
	_ = ctx // ctx carries span for downstream propagation
	span.SetAttributes(Attrs{
		AttrInputTokens:  100,
		AttrOutputTokens: 50,
	})
	span.SetStatus(StatusOK, "success")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "full-test" {
		t.Errorf("expected span name 'full-test', got %q", spans[0].Name)
	}
}
