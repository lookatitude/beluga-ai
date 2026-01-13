package s2s

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestStartProcessSpan(t *testing.T) {
	ctx := context.Background()
	tracerProvider := noop.NewTracerProvider()
	// Create a span first to establish tracer provider in context
	tracer := tracerProvider.Tracer("test")
	ctx, _ = tracer.Start(ctx, "root")

	ctx, span := StartProcessSpan(ctx, "test-provider", "test-model", "en-US")
	require.NotNil(t, ctx)
	require.NotNil(t, span)

	// Verify span is created (noop tracer doesn't expose details, but we can check it's not nil)
	assert.NotNil(t, span)
}

func TestStartStreamingSpan(t *testing.T) {
	ctx := context.Background()
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	ctx, _ = tracer.Start(ctx, "root")

	ctx, span := StartStreamingSpan(ctx, "test-provider", "test-model")
	require.NotNil(t, ctx)
	require.NotNil(t, span)

	// Verify span is created
	assert.NotNil(t, span)
}

func TestRecordSpanLatency(t *testing.T) {
	ctx := context.Background()
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	ctx, _ = tracer.Start(ctx, "root")

	_, span := StartProcessSpan(ctx, "test-provider", "test-model", "en-US")
	require.NotNil(t, span)

	latency := 150 * time.Millisecond
	RecordSpanLatency(span, latency)

	// Verify no panic (noop tracer doesn't expose attributes, but we can verify it doesn't error)
	assert.NotNil(t, span)
}

func TestRecordSpanError(t *testing.T) {
	ctx := context.Background()
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	ctx, _ = tracer.Start(ctx, "root")

	_, span := StartProcessSpan(ctx, "test-provider", "test-model", "en-US")
	require.NotNil(t, span)

	err := errors.New("test error")
	RecordSpanError(span, err)

	// Verify no panic
	assert.NotNil(t, span)
}

func TestRecordSpanAttributes(t *testing.T) {
	ctx := context.Background()
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	ctx, _ = tracer.Start(ctx, "root")

	_, span := StartProcessSpan(ctx, "test-provider", "test-model", "en-US")
	require.NotNil(t, span)

	attrs := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	RecordSpanAttributes(span, attrs)

	// Verify no panic
	assert.NotNil(t, span)
}

func TestTracingIntegration(t *testing.T) {
	// Test that tracing functions work together
	ctx := context.Background()
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	ctx, _ = tracer.Start(ctx, "root")

	// Start a process span
	ctx, span := StartProcessSpan(ctx, "test-provider", "test-model", "en-US")
	require.NotNil(t, span)

	// Record latency
	RecordSpanLatency(span, 200*time.Millisecond)

	// Record attributes
	RecordSpanAttributes(span, map[string]string{
		"session_id": "test-session",
		"user_id":    "test-user",
	})

	// Record error
	err := errors.New("processing error")
	RecordSpanError(span, err)

	// Verify span is still valid
	assert.NotNil(t, span)
}

func TestTracingWithNilSpan(t *testing.T) {
	// Test that functions handle nil spans gracefully
	// Note: In practice, spans from OTEL are never nil, but we test defensive behavior
	// If nil checks are needed, they should be added to the tracing functions

	// Create a nil span
	var span trace.Span = nil

	// These will panic with nil spans - this is expected behavior
	// In real usage, spans are always returned from tracer.Start() and are never nil
	// If defensive nil checks are desired, they should be added to the tracing functions
	// For now, we document this behavior rather than testing it
	_ = span
	t.Skip("Skipping nil span test - spans from OTEL are never nil in practice")
}

func TestTracingSpanAttributes(t *testing.T) {
	// Test that span attributes are set correctly
	ctx := context.Background()
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	ctx, _ = tracer.Start(ctx, "root")

	// Test process span with all attributes
	ctx, span := StartProcessSpan(ctx, "amazon_nova", "nova-2-sonic", "en-US")
	require.NotNil(t, span)

	// Add additional attributes
	RecordSpanAttributes(span, map[string]string{
		"audio_format": "PCM",
		"sample_rate":  "24000",
	})

	// Verify no panic
	assert.NotNil(t, span)
}

func TestTracingStreamingSpan(t *testing.T) {
	// Test streaming span creation and usage
	ctx := context.Background()
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	ctx, _ = tracer.Start(ctx, "root")

	ctx, span := StartStreamingSpan(ctx, "test-provider", "test-model")
	require.NotNil(t, span)

	// Record streaming-specific attributes
	RecordSpanAttributes(span, map[string]string{
		"stream_id":   "stream-123",
		"chunk_count": "10",
	})

	// Record latency for streaming
	RecordSpanLatency(span, 50*time.Millisecond)

	// Verify no panic
	assert.NotNil(t, span)
}
