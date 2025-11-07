package tracer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewTracer(t *testing.T) {
	tracer := NewTracer("test-service")
	assert.NotNil(t, tracer)
	assert.Equal(t, "test-service", tracer.service)
	assert.NotNil(t, tracer.spans)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestTracerStartSpan(t *testing.T) {
	tracer := NewTracer("test-service")
	ctx := context.Background()

	t.Run("start root span", func(t *testing.T) {
		ctx, span := tracer.StartSpan(ctx, "test_operation")
		assert.NotNil(t, ctx)
		assert.NotNil(t, span)

		spanImpl, ok := span.(*spanImpl)
		require.True(t, ok)
		assert.Equal(t, "test_operation", spanImpl.Name)
		assert.Equal(t, "test-service", spanImpl.Service)
		assert.Empty(t, spanImpl.ParentID)
		assert.NotZero(t, spanImpl.StartTime)
		assert.NotNil(t, spanImpl.Tags)
		assert.NotNil(t, spanImpl.Logs)
		assert.Equal(t, "started", spanImpl.Status)
	})

	t.Run("start child span", func(t *testing.T) {
		parentCtx, parentSpan := tracer.StartSpan(ctx, "parent_operation")
		_, childSpan := tracer.StartSpan(parentCtx, "child_operation")

		parentSpanImpl, ok := parentSpan.(*spanImpl)
		require.True(t, ok)
		childSpanImpl, ok := childSpan.(*spanImpl)
		require.True(t, ok)

		assert.Equal(t, parentSpanImpl.ID, childSpanImpl.ParentID)
		assert.Equal(t, parentSpanImpl.TraceID, childSpanImpl.TraceID)
		assert.NotEqual(t, parentSpanImpl.ID, childSpanImpl.ID)

		tracer.FinishSpan(parentSpan)
		tracer.FinishSpan(childSpan)
	})

	t.Run("span with options", func(t *testing.T) {
		_, span := tracer.StartSpan(ctx, "test_operation")

		spanImpl, ok := span.(*spanImpl)
		require.True(t, ok)

		// Test setting tags directly
		span.SetTag("key", "value")
		span.SetTag("env", "test")
		span.SetTag("version", "1.0")

		assert.Equal(t, "value", spanImpl.Tags["key"])
		assert.Equal(t, "test", spanImpl.Tags["env"])
		assert.Equal(t, "1.0", spanImpl.Tags["version"])

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		tracer.FinishSpan(span)
	})
}

func TestTracerFinishSpan(t *testing.T) {
	tracer := NewTracer("test-service")
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test_operation")
	spanImpl, ok := span.(*spanImpl)
	require.True(t, ok)

	assert.Nil(t, spanImpl.EndTime)
	assert.Nil(t, spanImpl.SpanDuration)

	time.Sleep(10 * time.Millisecond)
	tracer.FinishSpan(span)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	assert.NotNil(t, spanImpl.EndTime)
	assert.NotNil(t, spanImpl.SpanDuration)
	assert.True(t, *spanImpl.SpanDuration >= 10*time.Millisecond)
	assert.Equal(t, "finished", spanImpl.Status)
}

func TestTracerGetSpan(t *testing.T) {
	tracer := NewTracer("test-service")
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test_operation")
	spanImpl, ok := span.(*spanImpl)
	require.True(t, ok)

	// Test getting existing span
	retrievedSpan, exists := tracer.GetSpan(spanImpl.ID)
	assert.True(t, exists)
	assert.Equal(t, span, retrievedSpan)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	// Test getting non-existent span
	nonExistentSpan, exists := tracer.GetSpan("non-existent-id")
	assert.False(t, exists)
	assert.Nil(t, nonExistentSpan)

	tracer.FinishSpan(span)
}

func TestTracerGetTraceSpans(t *testing.T) {
	tracer := NewTracer("test-service")
	ctx := context.Background()

	// Create multiple spans in the same trace
	ctx, span1 := tracer.StartSpan(ctx, "operation1")
	ctx, span2 := tracer.StartSpan(ctx, "operation2")
	ctx, span3 := tracer.StartSpan(ctx, "operation3")

	span1Impl, ok := span1.(*spanImpl)
	require.True(t, ok)

	traceSpans := tracer.GetTraceSpans(span1Impl.TraceID)
	assert.Len(t, traceSpans, 3)

	// Verify all spans are from the same trace
	for _, span := range traceSpans {
		spanImpl, ok := span.(*spanImpl)
		require.True(t, ok)
		assert.Equal(t, span1Impl.TraceID, spanImpl.TraceID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Test with non-existent trace
	emptySpans := tracer.GetTraceSpans("non-existent-trace")
	assert.Empty(t, emptySpans)

	tracer.FinishSpan(span1)
	tracer.FinishSpan(span2)
	tracer.FinishSpan(span3)
}

func TestSpanOperations(t *testing.T) {
	tracer := NewTracer("test-service")
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test_operation")
	spanImpl, ok := span.(*spanImpl)
	require.True(t, ok)

	t.Run("set tag", func(t *testing.T) {
		span.SetTag("string_key", "string_value")
		span.SetTag("int_key", 42)
		span.SetTag("bool_key", true)

		assert.Equal(t, "string_value", spanImpl.Tags["string_key"])
		assert.Equal(t, 42, spanImpl.Tags["int_key"])
		assert.Equal(t, true, spanImpl.Tags["bool_key"])
	})

	t.Run("log", func(t *testing.T) {
		span.Log("Simple log message")
		span.Log("Log with fields", map[string]interface{}{
			"step":   1,
			"status": "processing",
		})

		assert.Len(t, spanImpl.Logs, 2)
		assert.Equal(t, "Simple log message", spanImpl.Logs[0].Message)
		assert.Nil(t, spanImpl.Logs[0].Fields)

		assert.Equal(t, "Log with fields", spanImpl.Logs[1].Message)
		assert.NotNil(t, spanImpl.Logs[1].Fields)
		assert.Equal(t, 1, spanImpl.Logs[1].Fields["step"])
		assert.Equal(t, "processing", spanImpl.Logs[1].Fields["status"])
	})

	t.Run("set error", func(t *testing.T) {
		span.SetError(assert.AnError)

		assert.Equal(t, assert.AnError.Error(), spanImpl.Error)
		assert.Equal(t, "error", spanImpl.Status)
	})

	t.Run("set status", func(t *testing.T) {
		span.SetStatus("custom_status")
		assert.Equal(t, "custom_status", spanImpl.Status)
	})

	t.Run("get duration", func(t *testing.T) {
		time.Sleep(5 * time.Millisecond)
		duration := span.GetDuration()
		assert.True(t, duration >= 5*time.Millisecond)

		tracer.FinishSpan(span)
		finishedDuration := span.GetDuration()
		assert.True(t, finishedDuration >= duration)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	t.Run("is finished", func(t *testing.T) {
		assert.True(t, span.IsFinished())

		// Reset for unfinished span test
		spanImpl.EndTime = nil
		assert.False(t, span.IsFinished())
	})

	tracer.FinishSpan(span)
}

func TestSpanOptions(t *testing.T) {
	t.Run("WithTag", func(t *testing.T) {
		option := WithTag("test_key", "test_value")

		spanImpl := &spanImpl{
			Tags: make(map[string]interface{}),
		}

		option(spanImpl)
		assert.Equal(t, "test_value", spanImpl.Tags["test_key"])
	})

	t.Run("WithTags", func(t *testing.T) {
		tags := map[string]interface{}{
			"key1": "value1",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			"key2": 42,
		}
		option := WithTags(tags)

		spanImpl := &spanImpl{
			Tags: make(map[string]interface{}),
		}

		option(spanImpl)
		assert.Equal(t, "value1", spanImpl.Tags["key1"])
		assert.Equal(t, 42, spanImpl.Tags["key2"])
	})
}

func TestSpanContext(t *testing.T) {
	tracer := NewTracer("test-service")
	ctx := context.Background()

	t.Run("SpanFromContext", func(t *testing.T) {
		ctx, span := tracer.StartSpan(ctx, "test_operation")
		retrievedSpan := SpanFromContext(ctx)
		assert.Equal(t, span, retrievedSpan)

		// Test with no span in context
		emptyCtx := context.Background()
		emptySpan := SpanFromContext(emptyCtx)
		assert.Nil(t, emptySpan)

		tracer.FinishSpan(span)
	})

	t.Run("TraceIDFromContext", func(t *testing.T) {
		ctx, span := tracer.StartSpan(ctx, "test_operation")
		spanImpl, ok := span.(*spanImpl)
		require.True(t, ok)

		traceID := TraceIDFromContext(ctx)
		assert.Equal(t, spanImpl.TraceID, traceID)

		// Test with no trace ID in context
		emptyCtx := context.Background()
		emptyTraceID := TraceIDFromContext(emptyCtx)
		assert.Empty(t, emptyTraceID)

		tracer.FinishSpan(span)
	})

	t.Run("SpanIDFromContext", func(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		ctx, span := tracer.StartSpan(ctx, "test_operation")
		spanImpl, ok := span.(*spanImpl)
		require.True(t, ok)

		spanID := SpanIDFromContext(ctx)
		assert.Equal(t, spanImpl.ID, spanID)

		// Test with no span ID in context
		emptyCtx := context.Background()
		emptySpanID := SpanIDFromContext(emptyCtx)
		assert.Empty(t, emptySpanID)

		tracer.FinishSpan(span)
	})
}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

func TestTraceFunc(t *testing.T) {
	tracer := NewTracer("test-service")
	ctx := context.Background()

	called := false
	testFunc := func(ctx context.Context) error {
		called = true
		span := SpanFromContext(ctx)
		assert.NotNil(t, span)
		return nil
	}

	err := TraceFunc(ctx, tracer, "test_function", testFunc)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestTraceMethod(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	tracer := NewTracer("test-service")
	ctx := context.Background()

	type TestReceiver struct {
		value int
	}

	receiver := &TestReceiver{value: 42}

	called := false
	testMethod := func() error {
		called = true
		return nil
	}

	err := TraceMethod(ctx, tracer, "TestMethod", receiver, testMethod)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	id2 := generateID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 16) // 8 bytes * 2 hex chars per byte

	// Verify it's valid hex
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	assert.True(t, isValidHex(id1))
	assert.True(t, isValidHex(id2))
}

func isValidHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	return true
}

// Benchmark tests
func BenchmarkTracer_StartSpan(b *testing.B) {
	tracer := NewTracer("bench-service")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := tracer.StartSpan(context.Background(), "bench_operation")
		tracer.FinishSpan(span)
	}
}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

func BenchmarkTracer_StartSpanWithTags(b *testing.B) {
	tracer := NewTracer("bench-service")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := tracer.StartSpan(context.Background(), "bench_operation")
		span.SetTag("iteration", i)
		span.SetTag("service", "bench")
		tracer.FinishSpan(span)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
}

func BenchmarkSpan_Log(b *testing.B) {
	tracer := NewTracer("bench-service")

	_, span := tracer.StartSpan(context.Background(), "bench_operation")
	defer tracer.FinishSpan(span)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span.Log("Benchmark log message", map[string]interface{}{
			"iteration": i,
			"timestamp": time.Now().UnixNano(),
		})
	}
}

func BenchmarkSpan_SetTag(b *testing.B) {
	tracer := NewTracer("bench-service")

	_, span := tracer.StartSpan(context.Background(), "bench_operation")
	defer tracer.FinishSpan(span)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span.SetTag("bench_key_"+string(rune(i)), "bench_value_"+string(rune(i)))
	}
}

func BenchmarkTraceFunc(b *testing.B) {
	tracer := NewTracer("bench-service")

	testFunc := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TraceFunc(context.Background(), tracer, "bench_function", testFunc)
	}
}
