package playground

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SpanExporter is a minimal OTel SDK SpanExporter that converts spans
// into [SpanEvent] values and pushes them onto the [Server]'s span
// sink. The devloop supervisor is expected to register it via an
// additional BatchSpanProcessor on the project's TracerProvider when
// `--playground` is enabled.
type SpanExporter struct {
	sink       chan<- SpanEvent
	restart    func() int
	mu         sync.Mutex
	shutdownCh chan struct{}
}

// NewSpanExporter returns a new exporter. The `restart` callback
// returns the current devloop restart sequence, so spans can be
// associated with the run they came from.
func NewSpanExporter(sink chan<- SpanEvent, restart func() int) *SpanExporter {
	return &SpanExporter{
		sink:       sink,
		restart:    restart,
		shutdownCh: make(chan struct{}),
	}
}

// ExportSpans implements sdktrace.SpanExporter. It never blocks the
// OTel pipeline — when the sink is full, spans are dropped rather
// than back-pressuring the scaffolded project's tracer.
func (e *SpanExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	if e.isShutdown() {
		return errors.New("playground: span exporter shut down")
	}
	seq := 0
	if e.restart != nil {
		seq = e.restart()
	}
	for _, sp := range spans {
		ev := spanToEvent(sp, seq)
		select {
		case e.sink <- ev:
		default:
		}
	}
	return nil
}

// Shutdown implements sdktrace.SpanExporter. It idempotently signals
// that no further spans should be accepted.
func (e *SpanExporter) Shutdown(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	select {
	case <-e.shutdownCh:
	default:
		close(e.shutdownCh)
	}
	return nil
}

func (e *SpanExporter) isShutdown() bool {
	select {
	case <-e.shutdownCh:
		return true
	default:
		return false
	}
}

func spanToEvent(sp sdktrace.ReadOnlySpan, seq int) SpanEvent {
	ev := SpanEvent{
		Name:       sp.Name(),
		StartTime:  sp.StartTime(),
		DurationMs: sp.EndTime().Sub(sp.StartTime()).Milliseconds(),
		RestartSeq: seq,
	}
	switch sp.Status().Code {
	case codes.Error:
		ev.Status = "error"
	case codes.Ok:
		ev.Status = "ok"
	default:
		ev.Status = "unset"
	}
	for _, a := range sp.Attributes() {
		switch a.Key {
		case "gen_ai.request.model", "gen_ai.response.model":
			if ev.Model == "" {
				ev.Model = stringValue(a.Value)
			}
		case "gen_ai.usage.input_tokens", "gen_ai.usage.prompt_tokens":
			if ev.InputTokens == 0 {
				ev.InputTokens = int64Value(a.Value)
			}
		case "gen_ai.usage.output_tokens", "gen_ai.usage.completion_tokens":
			if ev.OutputTokens == 0 {
				ev.OutputTokens = int64Value(a.Value)
			}
		}
	}
	return ev
}

func stringValue(v attribute.Value) string {
	switch v.Type() {
	case attribute.STRING:
		return v.AsString()
	default:
		return v.Emit()
	}
}

func int64Value(v attribute.Value) int64 {
	switch v.Type() {
	case attribute.INT64:
		return v.AsInt64()
	case attribute.FLOAT64:
		return int64(v.AsFloat64())
	default:
		return 0
	}
}
