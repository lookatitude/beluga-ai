package transport

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

func TestMetrics_RecordConnection(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordConnection(ctx, "webrtc", 100*time.Millisecond, true)
	metrics.RecordConnection(ctx, "webrtc", 100*time.Millisecond, false)
}

func TestMetrics_RecordDisconnection(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordDisconnection(ctx, "webrtc", 50*time.Millisecond)
}

func TestMetrics_RecordAudio(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordAudioSent(ctx, "webrtc", 1024)
	metrics.RecordAudioReceived(ctx, "webrtc", 2048)
}

func TestMetrics_RecordError(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordError(ctx, "webrtc", ErrCodeConnectionFailed, 100*time.Millisecond)
}

func TestMetrics_Connections(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.IncrementConnections(ctx, "webrtc")
	metrics.DecrementConnections(ctx, "webrtc")
}

func TestNoOpMetrics(t *testing.T) {
	noOp := NoOpMetrics()
	ctx := context.Background()

	// Should not panic
	noOp.RecordConnection(ctx, "webrtc", 100*time.Millisecond, true)
	noOp.RecordDisconnection(ctx, "webrtc", 50*time.Millisecond)
	noOp.RecordAudioSent(ctx, "webrtc", 1024)
	noOp.RecordAudioReceived(ctx, "webrtc", 2048)
	noOp.RecordError(ctx, "webrtc", ErrCodeConnectionFailed, 100*time.Millisecond)
	noOp.IncrementConnections(ctx, "webrtc")
	noOp.DecrementConnections(ctx, "webrtc")
}
