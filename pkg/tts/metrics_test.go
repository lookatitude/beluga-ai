package tts

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

func TestMetrics_RecordGeneration(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordGeneration(ctx, "openai", "tts-1", "nova", 100*time.Millisecond)
}

func TestMetrics_RecordError(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordError(ctx, "openai", "tts-1", "nova", ErrCodeNetworkError, 100*time.Millisecond)
}

func TestMetrics_RecordStreaming(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordStreaming(ctx, "openai", "tts-1", "nova", 100*time.Millisecond)
}

func TestMetrics_ActiveStreams(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.IncrementActiveStreams(ctx, "openai", "tts-1", "nova")
	metrics.DecrementActiveStreams(ctx, "openai", "tts-1", "nova")
}

func TestNoOpMetrics(t *testing.T) {
	noOp := NoOpMetrics()
	ctx := context.Background()

	// Should not panic
	noOp.RecordGeneration(ctx, "openai", "tts-1", "nova", 100*time.Millisecond)
	noOp.RecordError(ctx, "openai", "tts-1", "nova", ErrCodeNetworkError, 100*time.Millisecond)
	noOp.RecordStreaming(ctx, "openai", "tts-1", "nova", 100*time.Millisecond)
	noOp.IncrementActiveStreams(ctx, "openai", "tts-1", "nova")
	noOp.DecrementActiveStreams(ctx, "openai", "tts-1", "nova")
}
