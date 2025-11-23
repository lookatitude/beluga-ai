package vad

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

func TestMetrics_RecordProcessing(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.RecordProcessing(ctx, "silero", 10*time.Millisecond, true)
	metrics.RecordProcessing(ctx, "silero", 10*time.Millisecond, false)
}

func TestMetrics_RecordError(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.RecordError(ctx, "silero", ErrCodeProcessingError, 10*time.Millisecond)
}

func TestMetrics_IncrementProcessedFrames(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.IncrementProcessedFrames(ctx, "silero")
}

func TestNoOpMetrics(t *testing.T) {
	noOp := NewNoOpMetrics()
	ctx := context.Background()

	// Should not panic
	noOp.RecordProcessing(ctx, "silero", 10*time.Millisecond, true)
	noOp.RecordError(ctx, "silero", ErrCodeProcessingError, 10*time.Millisecond)
	noOp.IncrementProcessedFrames(ctx, "silero")
}
