package stt

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

func TestMetrics_RecordTranscription(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordTranscription(ctx, "deepgram", "nova-3", 100*time.Millisecond)
}

func TestMetrics_RecordError(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordError(ctx, "deepgram", "nova-3", ErrCodeNetworkError, 100*time.Millisecond)
}

func TestMetrics_RecordStreaming(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordStreaming(ctx, "deepgram", "nova-3", 100*time.Millisecond)
}

func TestMetrics_ActiveStreams(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.IncrementActiveStreams(ctx, "deepgram", "nova-3")
	metrics.DecrementActiveStreams(ctx, "deepgram", "nova-3")
}

func TestNoOpMetrics(t *testing.T) {
	noOp := NoOpMetrics()
	ctx := context.Background()

	// Should not panic
	noOp.RecordTranscription(ctx, "deepgram", "nova-3", 100*time.Millisecond)
	noOp.RecordError(ctx, "deepgram", "nova-3", ErrCodeNetworkError, 100*time.Millisecond)
	noOp.RecordStreaming(ctx, "deepgram", "nova-3", 100*time.Millisecond)
	noOp.IncrementActiveStreams(ctx, "deepgram", "nova-3")
	noOp.DecrementActiveStreams(ctx, "deepgram", "nova-3")
}
