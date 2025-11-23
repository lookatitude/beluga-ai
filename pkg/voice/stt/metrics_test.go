package stt

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestNewMetrics(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	assert.NotNil(t, metrics)
}

func TestMetrics_RecordTranscription(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.RecordTranscription(ctx, "deepgram", "nova-3", 100*time.Millisecond)
}

func TestMetrics_RecordError(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.RecordError(ctx, "deepgram", "nova-3", ErrCodeNetworkError, 100*time.Millisecond)
}

func TestMetrics_RecordStreaming(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.RecordStreaming(ctx, "deepgram", "nova-3", 100*time.Millisecond)
}

func TestMetrics_ActiveStreams(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.IncrementActiveStreams(ctx, "deepgram", "nova-3")
	metrics.DecrementActiveStreams(ctx, "deepgram", "nova-3")
}

func TestNoOpMetrics(t *testing.T) {
	noOp := NewNoOpMetrics()
	ctx := context.Background()

	// Should not panic
	noOp.RecordTranscription(ctx, "deepgram", "nova-3", 100*time.Millisecond)
	noOp.RecordError(ctx, "deepgram", "nova-3", ErrCodeNetworkError, 100*time.Millisecond)
	noOp.RecordStreaming(ctx, "deepgram", "nova-3", 100*time.Millisecond)
	noOp.IncrementActiveStreams(ctx, "deepgram", "nova-3")
	noOp.DecrementActiveStreams(ctx, "deepgram", "nova-3")
}
