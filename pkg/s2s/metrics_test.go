package s2s

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func TestNewMetrics(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	assert.NotNil(t, metrics)
}

func TestMetrics_RecordProcess(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordProcess(ctx, "amazon_nova", "nova-2-sonic", 100*time.Millisecond)
}

func TestMetrics_RecordError(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordError(ctx, "amazon_nova", "nova-2-sonic", ErrCodeNetworkError, 100*time.Millisecond)
}

func TestMetrics_RecordStreaming(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordStreaming(ctx, "amazon_nova", "nova-2-sonic", 100*time.Millisecond)
}

func TestMetrics_ActiveStreams(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.IncrementActiveStreams(ctx, "amazon_nova", "nova-2-sonic")
	metrics.DecrementActiveStreams(ctx, "amazon_nova", "nova-2-sonic")
}

func TestMetrics_ProviderUsage(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordProviderUsage(ctx, "amazon_nova")
}

func TestMetrics_Fallback(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordFallback(ctx, "amazon_nova", "grok")
}

func TestMetrics_ConcurrentSessions(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordConcurrentSessions(ctx, "amazon_nova", 10)
}

func TestMetrics_ReasoningMode(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordReasoningMode(ctx, "amazon_nova", "built-in")
	metrics.RecordReasoningMode(ctx, "grok", "external")
}

func TestMetrics_LatencyTarget(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordLatencyTarget(ctx, "amazon_nova", "low", 50*time.Millisecond)
	metrics.RecordLatencyTarget(ctx, "grok", "medium", 200*time.Millisecond)
	metrics.RecordLatencyTarget(ctx, "gemini", "high", 500*time.Millisecond)
}

func TestMetrics_AudioQuality(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordAudioQuality(ctx, "amazon_nova", 0.95)
	metrics.RecordAudioQuality(ctx, "grok", 0.88)
	metrics.RecordAudioQuality(ctx, "gemini", 0.92)
}

func TestNoOpMetrics(t *testing.T) {
	noOp := NoOpMetrics()
	ctx := context.Background()

	// Should not panic
	noOp.RecordProcess(ctx, "amazon_nova", "nova-2-sonic", 100*time.Millisecond)
	noOp.RecordError(ctx, "amazon_nova", "nova-2-sonic", ErrCodeNetworkError, 100*time.Millisecond)
	noOp.RecordStreaming(ctx, "amazon_nova", "nova-2-sonic", 100*time.Millisecond)
	noOp.IncrementActiveStreams(ctx, "amazon_nova", "nova-2-sonic")
	noOp.DecrementActiveStreams(ctx, "amazon_nova", "nova-2-sonic")
	noOp.RecordProviderUsage(ctx, "amazon_nova")
	noOp.RecordFallback(ctx, "amazon_nova", "grok")
	noOp.RecordConcurrentSessions(ctx, "amazon_nova", 10)
	noOp.RecordReasoningMode(ctx, "amazon_nova", "built-in")
	noOp.RecordLatencyTarget(ctx, "amazon_nova", "low", 50*time.Millisecond)
	noOp.RecordAudioQuality(ctx, "amazon_nova", 0.95)
}
