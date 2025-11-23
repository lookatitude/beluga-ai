package noise

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
	metrics.RecordProcessing(ctx, "rnnoise", 10*time.Millisecond, 1024, 1024)
}

func TestMetrics_RecordError(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.RecordError(ctx, "rnnoise", ErrCodeProcessingError, 10*time.Millisecond)
}

func TestMetrics_IncrementProcessedFrames(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.IncrementProcessedFrames(ctx, "rnnoise")
}

func TestNoOpMetrics(t *testing.T) {
	noOp := NewNoOpMetrics()
	ctx := context.Background()

	// Should not panic
	noOp.RecordProcessing(ctx, "rnnoise", 10*time.Millisecond, 1024, 1024)
	noOp.RecordError(ctx, "rnnoise", ErrCodeProcessingError, 10*time.Millisecond)
	noOp.IncrementProcessedFrames(ctx, "rnnoise")
}
