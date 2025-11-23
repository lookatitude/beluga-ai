package turndetection

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

func TestMetrics_RecordDetection(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.RecordDetection(ctx, "heuristic", 10*time.Millisecond, true)
	metrics.RecordDetection(ctx, "heuristic", 10*time.Millisecond, false)
}

func TestMetrics_RecordError(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.RecordError(ctx, "heuristic", ErrCodeProcessingError, 10*time.Millisecond)
}

func TestMetrics_IncrementDetections(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.IncrementDetections(ctx, "heuristic")
}

func TestNoOpMetrics(t *testing.T) {
	noOp := NewNoOpMetrics()
	ctx := context.Background()

	// Should not panic
	noOp.RecordDetection(ctx, "heuristic", 10*time.Millisecond, true)
	noOp.RecordError(ctx, "heuristic", ErrCodeProcessingError, 10*time.Millisecond)
	noOp.IncrementDetections(ctx, "heuristic")
}
