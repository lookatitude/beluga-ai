package playground

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// spanRecord is a minimal in-test implementation of
// sdktrace.ReadOnlySpan. Production code uses the real sdktrace spans
// produced by a TracerProvider; this shape lets us exercise
// spanToEvent/stringValue/int64Value deterministically.
type spanRecord struct {
	name       string
	start, end time.Time
	status     sdktrace.Status
	attrs      []attribute.KeyValue
	sdktrace.ReadOnlySpan
}

func (s spanRecord) Name() string         { return s.name }
func (s spanRecord) StartTime() time.Time { return s.start }
func (s spanRecord) EndTime() time.Time   { return s.end }
func (s spanRecord) Attributes() []attribute.KeyValue {
	return s.attrs
}
func (s spanRecord) Status() sdktrace.Status { return s.status }
func (s spanRecord) SpanContext() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{})
}

func TestSpanToEvent_PopulatesGenAIAttributes(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	ev := spanToEvent(spanRecord{
		name:   "llm.chat",
		start:  start,
		end:    start.Add(42 * time.Millisecond),
		status: sdktrace.Status{Code: codes.Ok},
		attrs: []attribute.KeyValue{
			attribute.String("gen_ai.request.model", "gpt-5"),
			attribute.Int64("gen_ai.usage.input_tokens", 120),
			attribute.Int64("gen_ai.usage.output_tokens", 80),
		},
	}, 7)
	if ev.Name != "llm.chat" {
		t.Errorf("name = %q", ev.Name)
	}
	if ev.DurationMs != 42 {
		t.Errorf("duration = %d", ev.DurationMs)
	}
	if ev.Status != "ok" {
		t.Errorf("status = %q want ok", ev.Status)
	}
	if ev.Model != "gpt-5" {
		t.Errorf("model = %q", ev.Model)
	}
	if ev.InputTokens != 120 || ev.OutputTokens != 80 {
		t.Errorf("tokens in=%d out=%d", ev.InputTokens, ev.OutputTokens)
	}
	if ev.RestartSeq != 7 {
		t.Errorf("restart_seq = %d", ev.RestartSeq)
	}
}

func TestSpanToEvent_StatusMapping(t *testing.T) {
	cases := []struct {
		in   codes.Code
		want string
	}{
		{codes.Ok, "ok"},
		{codes.Error, "error"},
		{codes.Unset, "unset"},
	}
	for _, c := range cases {
		ev := spanToEvent(spanRecord{
			name:   "n",
			status: sdktrace.Status{Code: c.in},
		}, 0)
		if ev.Status != c.want {
			t.Errorf("code=%v → status=%q want %q", c.in, ev.Status, c.want)
		}
	}
}

func TestSpanToEvent_FallbackModelAndTokens(t *testing.T) {
	// response.model is accepted when request.model is missing; prompt
	// tokens map from the legacy attribute name; completion_tokens too.
	ev := spanToEvent(spanRecord{
		name: "llm.legacy",
		attrs: []attribute.KeyValue{
			attribute.String("gen_ai.response.model", "legacy-1"),
			attribute.Int64("gen_ai.usage.prompt_tokens", 5),
			attribute.Int64("gen_ai.usage.completion_tokens", 9),
		},
	}, 0)
	if ev.Model != "legacy-1" {
		t.Errorf("model = %q", ev.Model)
	}
	if ev.InputTokens != 5 || ev.OutputTokens != 9 {
		t.Errorf("tokens in=%d out=%d", ev.InputTokens, ev.OutputTokens)
	}
}

func TestStringValue_NonStringFallsBackToEmit(t *testing.T) {
	if got := stringValue(attribute.StringValue("hi")); got != "hi" {
		t.Errorf("string passthrough = %q", got)
	}
	if got := stringValue(attribute.Int64Value(42)); got != "42" {
		t.Errorf("int emit = %q", got)
	}
}

func TestInt64Value_Conversions(t *testing.T) {
	if got := int64Value(attribute.Int64Value(17)); got != 17 {
		t.Errorf("int64 = %d", got)
	}
	if got := int64Value(attribute.Float64Value(3.7)); got != 3 {
		t.Errorf("float64 truncation = %d want 3", got)
	}
	if got := int64Value(attribute.StringValue("x")); got != 0 {
		t.Errorf("non-numeric = %d want 0", got)
	}
}

func TestSpanExporter_ExportSpansPushesToSink(t *testing.T) {
	sink := make(chan SpanEvent, 4)
	exp := NewSpanExporter(sink, func() int { return 3 })

	start := time.Now()
	spans := []sdktrace.ReadOnlySpan{
		spanRecord{
			name: "s1", start: start, end: start.Add(time.Millisecond),
			status: sdktrace.Status{Code: codes.Ok},
		},
		spanRecord{
			name: "s2", start: start, end: start.Add(time.Millisecond),
		},
	}
	if err := exp.ExportSpans(context.Background(), spans); err != nil {
		t.Fatalf("ExportSpans: %v", err)
	}
	for i, want := range []string{"s1", "s2"} {
		select {
		case got := <-sink:
			if got.Name != want {
				t.Errorf("span[%d] = %q want %q", i, got.Name, want)
			}
			if got.RestartSeq != 3 {
				t.Errorf("span[%d].restart_seq = %d want 3", i, got.RestartSeq)
			}
		case <-time.After(time.Second):
			t.Fatalf("sink never received span[%d]=%s", i, want)
		}
	}
}

func TestSpanExporter_ExportSpansDropsWhenSinkFull(t *testing.T) {
	sink := make(chan SpanEvent, 1)
	exp := NewSpanExporter(sink, nil)

	start := time.Now()
	spans := []sdktrace.ReadOnlySpan{
		spanRecord{name: "first", start: start, end: start},
		spanRecord{name: "dropped", start: start, end: start},
		spanRecord{name: "also-dropped", start: start, end: start},
	}
	if err := exp.ExportSpans(context.Background(), spans); err != nil {
		t.Fatalf("ExportSpans: %v", err)
	}
	// Sink has capacity 1 and was never drained, so only the first span
	// must be queued — the other two are deliberately dropped rather
	// than back-pressuring the OTel pipeline.
	select {
	case got := <-sink:
		if got.Name != "first" {
			t.Errorf("first queued = %q want first", got.Name)
		}
	default:
		t.Fatal("sink missing first span")
	}
	select {
	case got := <-sink:
		t.Errorf("unexpected extra span %q", got.Name)
	default:
	}
}

func TestSpanExporter_ShutdownIdempotentAndBlocksExport(t *testing.T) {
	sink := make(chan SpanEvent, 2)
	exp := NewSpanExporter(sink, nil)
	if err := exp.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown#1: %v", err)
	}
	// Second call must not panic on close(nil-channel) etc.
	if err := exp.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown#2: %v", err)
	}

	err := exp.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{
		spanRecord{name: "after-shutdown"},
	})
	if err == nil {
		t.Fatal("ExportSpans after shutdown must error")
	}
	var want *exportErr
	_ = errors.As(err, &want) // type-match lenient — we only care it's non-nil
}

// exportErr is a sentinel type used only to widen errors.As in the
// shutdown test above; the real exporter returns a fmt-wrapped error.
type exportErr struct{}

func (e *exportErr) Error() string { return "" }
