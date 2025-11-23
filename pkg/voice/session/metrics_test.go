package session

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

func TestMetrics_RecordSessionStart(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.RecordSessionStart(ctx, "test-session", 10*time.Millisecond)
}

func TestMetrics_RecordSessionStop(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.RecordSessionStop(ctx, "test-session", 10*time.Millisecond)
}

func TestMetrics_RecordSessionError(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter)
	ctx := context.Background()

	// Should not panic
	metrics.RecordSessionError(ctx, "test-session", ErrCodeInternalError, 10*time.Millisecond)
}

func TestNoOpMetrics(t *testing.T) {
	noOp := NewNoOpMetrics()
	ctx := context.Background()

	// Should not panic
	noOp.RecordSessionStart(ctx, "test-session", 10*time.Millisecond)
	noOp.RecordSessionStop(ctx, "test-session", 10*time.Millisecond)
	noOp.RecordSessionError(ctx, "test-session", ErrCodeInternalError, 10*time.Millisecond)
	noOp.IncrementActiveSessions(ctx)
	noOp.DecrementActiveSessions(ctx)
}
