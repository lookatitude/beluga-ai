// Package memory provides comprehensive tests for observability and monitoring functionality.
package memory

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// MockMeterProvider provides a mock implementation for testing
type MockMeterProvider struct {
	meter metric.Meter
}

func NewMockMeterProvider() *MockMeterProvider {
	return &MockMeterProvider{
		meter: noop.NewMeterProvider().Meter("test"),
	}
}

func (m *MockMeterProvider) Meter() metric.Meter {
	return m.meter
}

// MockTracerProvider provides a mock implementation for testing
type MockTracerProvider struct {
	tracer trace.Tracer
}

func NewMockTracerProvider() *MockTracerProvider {
	return &MockTracerProvider{
		tracer: otel.Tracer("test"),
	}
}

func (m *MockTracerProvider) Tracer() trace.Tracer {
	return m.tracer
}

// TestNewMetrics tests the Metrics constructor
func TestNewMetrics(t *testing.T) {
	mockMeter := NewMockMeterProvider().Meter()
	metrics := NewMetrics(mockMeter)

	assert.NotNil(t, metrics)
	assert.Equal(t, mockMeter, metrics.meter)
	assert.NotNil(t, metrics.loadDuration)
	assert.NotNil(t, metrics.saveDuration)
	assert.NotNil(t, metrics.clearDuration)
	assert.NotNil(t, metrics.operationCounter)
	assert.NotNil(t, metrics.errorCounter)
	assert.NotNil(t, metrics.memorySizeGauge)
	assert.NotNil(t, metrics.activeMemoryGauge)
}

// TestMetrics_RecordOperation tests recording operations
func TestMetrics_RecordOperation(t *testing.T) {
	ctx := context.Background()
	mockMeter := NewMockMeterProvider().Meter()
	metrics := NewMetrics(mockMeter)

	// Test successful operation
	metrics.RecordOperation(ctx, "load", MemoryTypeBuffer, true)

	// Test failed operation
	metrics.RecordOperation(ctx, "save", MemoryTypeBufferWindow, false)

	// Test with nil operation counter (should not panic)
	metrics.operationCounter = nil
	metrics.RecordOperation(ctx, "clear", MemoryTypeBuffer, true)
}

// TestMetrics_RecordOperationDuration tests recording operation durations
func TestMetrics_RecordOperationDuration(t *testing.T) {
	ctx := context.Background()
	mockMeter := NewMockMeterProvider().Meter()
	metrics := NewMetrics(mockMeter)

	duration := 150 * time.Millisecond

	// Test different operations
	metrics.RecordOperationDuration(ctx, "load", MemoryTypeBuffer, duration)
	metrics.RecordOperationDuration(ctx, "save", MemoryTypeBufferWindow, duration)
	metrics.RecordOperationDuration(ctx, "clear", MemoryTypeSummary, duration)

	// Test unknown operation (should not record)
	metrics.RecordOperationDuration(ctx, "unknown", MemoryTypeBuffer, duration)

	// Test with nil histogram (should not panic)
	metrics.loadDuration = nil
	metrics.RecordOperationDuration(ctx, "load", MemoryTypeBuffer, duration)
}

// TestMetrics_RecordError tests recording errors
func TestMetrics_RecordError(t *testing.T) {
	ctx := context.Background()
	mockMeter := NewMockMeterProvider().Meter()
	metrics := NewMetrics(mockMeter)

	// Test recording different error types
	metrics.RecordError(ctx, "load", MemoryTypeBuffer, "timeout")
	metrics.RecordError(ctx, "save", MemoryTypeBufferWindow, "storage_error")
	metrics.RecordError(ctx, "clear", MemoryTypeSummary, "validation_error")

	// Test with nil error counter (should not panic)
	metrics.errorCounter = nil
	metrics.RecordError(ctx, "load", MemoryTypeBuffer, "test_error")
}

// TestMetrics_RecordMemorySize tests recording memory size
func TestMetrics_RecordMemorySize(t *testing.T) {
	ctx := context.Background()
	mockMeter := NewMockMeterProvider().Meter()
	metrics := NewMetrics(mockMeter)

	// Test recording different sizes
	metrics.RecordMemorySize(ctx, MemoryTypeBuffer, 100)
	metrics.RecordMemorySize(ctx, MemoryTypeBufferWindow, 50)
	metrics.RecordMemorySize(ctx, MemoryTypeSummary, 200)

	// Test with nil gauge (should not panic)
	metrics.memorySizeGauge = nil
	metrics.RecordMemorySize(ctx, MemoryTypeBuffer, 100)
}

// TestMetrics_RecordActiveMemory tests recording active memory instances
func TestMetrics_RecordActiveMemory(t *testing.T) {
	ctx := context.Background()
	mockMeter := NewMockMeterProvider().Meter()
	metrics := NewMetrics(mockMeter)

	// Test recording creation and deletion
	metrics.RecordActiveMemory(ctx, MemoryTypeBuffer, 1)       // Created
	metrics.RecordActiveMemory(ctx, MemoryTypeBufferWindow, 1) // Created
	metrics.RecordActiveMemory(ctx, MemoryTypeBuffer, -1)      // Deleted

	// Test with nil gauge (should not panic)
	metrics.activeMemoryGauge = nil
	metrics.RecordActiveMemory(ctx, MemoryTypeBuffer, 1)
}

// TestNewTracer tests the Tracer constructor
func TestNewTracer(t *testing.T) {
	tracer := NewTracer()
	assert.NotNil(t, tracer)
	assert.NotNil(t, tracer.tracer)
}

// TestTracer_StartSpan tests starting spans
func TestTracer_StartSpan(t *testing.T) {
	ctx := context.Background()
	tracer := NewTracer()

	// Test starting spans for different operations
	ctx1, span1 := tracer.StartSpan(ctx, "load", MemoryTypeBuffer, "history")
	assert.NotNil(t, span1)
	span1.End()

	ctx2, span2 := tracer.StartSpan(ctx, "save", MemoryTypeBufferWindow, "recent")
	assert.NotNil(t, span2)
	span2.End()

	// Verify context is updated
	assert.NotEqual(t, ctx, ctx1)
	assert.NotEqual(t, ctx, ctx2)
}

// TestTracer_RecordSpanError tests recording errors on spans
func TestTracer_RecordSpanError(t *testing.T) {
	ctx := context.Background()
	tracer := NewTracer()

	_, span := tracer.StartSpan(ctx, "test", MemoryTypeBuffer, "test_memory")

	// Test recording error
	testErr := errors.New("test error")
	tracer.RecordSpanError(span, testErr)

	// Test recording nil error (should not panic)
	tracer.RecordSpanError(span, nil)

	span.End()
}

// TestGlobalMetricsFunctions tests global metrics functions
func TestGlobalMetricsFunctions(t *testing.T) {
	// Initially should be nil
	assert.Nil(t, GetGlobalMetrics())

	// Set global metrics
	mockMeter := NewMockMeterProvider().Meter()
	SetGlobalMetrics(mockMeter)

	// Should now return the metrics instance
	globalMetrics := GetGlobalMetrics()
	assert.NotNil(t, globalMetrics)
	assert.Equal(t, mockMeter, globalMetrics.meter)
}

// TestGlobalTracerFunctions tests global tracer functions
func TestGlobalTracerFunctions(t *testing.T) {
	// Tracer is initialized in init(), so it should not be nil
	// (The init() function calls SetGlobalTracer())
	assert.NotNil(t, GetGlobalTracer())

	// Set global tracer
	SetGlobalTracer()

	// Should now return the tracer instance
	globalTracer := GetGlobalTracer()
	assert.NotNil(t, globalTracer)
	assert.NotNil(t, globalTracer.tracer)
}

// TestNewLogger tests the Logger constructor
func TestNewLogger(t *testing.T) {
	// Test with nil logger
	logger := NewLogger(nil)
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.logger)

	// Test with custom logger
	customLogger := slog.New(slog.NewTextHandler(&strings.Builder{}, nil))
	logger2 := NewLogger(customLogger)
	assert.NotNil(t, logger2)
	assert.Equal(t, customLogger, logger2.logger)
}

// TestLogger_LogMemoryOperation tests logging memory operations
func TestLogger_LogMemoryOperation(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger(nil)

	duration := 100 * time.Millisecond

	// Test successful operation
	logger.LogMemoryOperation(ctx, slog.LevelInfo, "load", MemoryTypeBuffer, "history", 5, duration, nil)

	// Test operation with error
	testErr := errors.New("test error")
	logger.LogMemoryOperation(ctx, slog.LevelError, "save", MemoryTypeBufferWindow, "recent", 3, duration, testErr)
}

// TestLogger_LogMemoryLifecycle tests logging memory lifecycle events
func TestLogger_LogMemoryLifecycle(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger(nil)

	// Test logging lifecycle events
	logger.LogMemoryLifecycle(ctx, "created", MemoryTypeBuffer, "history")
	logger.LogMemoryLifecycle(ctx, "cleared", MemoryTypeBufferWindow, "recent",
		slog.String("reason", "user_request"),
		slog.Int("messages_cleared", 10))
}

// TestLogger_LogError tests logging errors
func TestLogger_LogError(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger(nil)

	testErr := errors.New("test error")

	// Test logging errors
	logger.LogError(ctx, testErr, "load", MemoryTypeBuffer, "history")
	logger.LogError(ctx, testErr, "save", MemoryTypeBufferWindow, "recent",
		slog.String("details", "additional context"),
		slog.Int("attempt", 2))
}

// TestGlobalLoggerFunctions tests global logger functions
func TestGlobalLoggerFunctions(t *testing.T) {
	// Initially should have a default logger
	globalLogger := GetGlobalLogger()
	assert.NotNil(t, globalLogger)

	// Set custom logger
	customLogger := NewLogger(slog.New(slog.NewTextHandler(&strings.Builder{}, nil)))
	SetGlobalLogger(customLogger)

	// Should return the custom logger
	assert.Equal(t, customLogger, GetGlobalLogger())
}

// TestConvenienceLoggingFunctions tests the convenience logging functions
func TestConvenienceLoggingFunctions(t *testing.T) {
	ctx := context.Background()

	// Reset global logger to ensure we have one
	SetGlobalLogger(NewLogger(nil))

	duration := 50 * time.Millisecond
	testErr := errors.New("convenience test error")

	// Test convenience functions
	LogMemoryOperation(ctx, slog.LevelInfo, "test_op", MemoryTypeBuffer, "test_memory", 2, duration, nil)
	LogMemoryOperation(ctx, slog.LevelWarn, "test_op_error", MemoryTypeBufferWindow, "test_memory", 1, duration, testErr)

	LogMemoryLifecycle(ctx, "test_event", MemoryTypeSummary, "test_memory")
	LogMemoryLifecycle(ctx, "test_event_attrs", MemoryTypeVectorStore, "test_memory",
		slog.String("custom_attr", "custom_value"))

	LogError(ctx, testErr, "test_error_op", MemoryTypeBuffer, "test_memory")
	LogError(ctx, testErr, "test_error_op_attrs", MemoryTypeBufferWindow, "test_memory",
		slog.String("error_details", "additional info"))
}

// TestMetricsIntegration_WithMemoryOperations tests metrics integration with actual memory operations
func TestMetricsIntegration_WithMemoryOperations(t *testing.T) {
	ctx := context.Background()

	// Set up global metrics
	mockMeter := NewMockMeterProvider().Meter()
	SetGlobalMetrics(mockMeter)

	// Create memory instance
	memory, err := NewMemory(MemoryTypeBuffer)
	require.NoError(t, err)

	// Perform operations that should trigger metrics
	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	start := time.Now()
	err = memory.SaveContext(ctx, inputs, outputs)
	duration := time.Since(start)
	assert.NoError(t, err)

	_, err = memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)

	err = memory.Clear(ctx)
	assert.NoError(t, err)

	// Verify global metrics instance is set
	globalMetrics := GetGlobalMetrics()
	assert.NotNil(t, globalMetrics)
	assert.Equal(t, mockMeter, globalMetrics.meter)

	// Test manual metrics recording
	globalMetrics.RecordOperation(ctx, "test_operation", MemoryTypeBuffer, true)
	globalMetrics.RecordOperationDuration(ctx, "save", MemoryTypeBuffer, duration)
	globalMetrics.RecordError(ctx, "test_error", MemoryTypeBuffer, "validation_error")
	globalMetrics.RecordMemorySize(ctx, MemoryTypeBuffer, 100)
	globalMetrics.RecordActiveMemory(ctx, MemoryTypeBuffer, 1)
}

// TestTracingIntegration_WithMemoryOperations tests tracing integration with memory operations
func TestTracingIntegration_WithMemoryOperations(t *testing.T) {
	ctx := context.Background()

	// Set up global tracer
	SetGlobalTracer()

	// Create memory instance
	memory, err := NewMemory(MemoryTypeBuffer)
	require.NoError(t, err)

	// Perform operations that should be traced
	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err = memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)

	_, err = memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)

	err = memory.Clear(ctx)
	assert.NoError(t, err)

	// Verify global tracer instance is set
	globalTracer := GetGlobalTracer()
	assert.NotNil(t, globalTracer)
	assert.NotNil(t, globalTracer.tracer)

	// Test manual tracing
	ctx, span := globalTracer.StartSpan(ctx, "manual_test", MemoryTypeBuffer, "test_memory")
	assert.NotNil(t, span)

	testErr := errors.New("test error")
	globalTracer.RecordSpanError(span, testErr)

	span.End()
}

// TestObservabilityIntegration_EndToEnd tests end-to-end observability integration
func TestObservabilityIntegration_EndToEnd(t *testing.T) {
	ctx := context.Background()

	// Set up global observability
	mockMeter := NewMockMeterProvider().Meter()
	SetGlobalMetrics(mockMeter)
	SetGlobalTracer()
	SetGlobalLogger(NewLogger(nil))

	// Create factory and memory instances
	factory := NewFactory()

	configs := []Config{
		{Type: MemoryTypeBuffer, Enabled: true, MemoryKey: "buffer_memory"},
		{Type: MemoryTypeBufferWindow, Enabled: true, MemoryKey: "window_memory", WindowSize: 3},
	}

	memories := make(map[string]Memory)

	for _, config := range configs {
		memory, err := factory.CreateMemory(ctx, config)
		require.NoError(t, err)
		memories[string(config.Type)] = memory
	}

	// Perform operations and verify observability
	for name, memory := range memories {
		t.Run("Observability_"+name, func(t *testing.T) {
			// Save context
			inputs := map[string]any{"input": "Test input"}
			outputs := map[string]any{"output": "Test output"}

			err := memory.SaveContext(ctx, inputs, outputs)
			assert.NoError(t, err)

			// Load memory variables
			vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
			assert.NoError(t, err)
			assert.Contains(t, vars, memory.MemoryVariables()[0])

			// Clear memory
			err = memory.Clear(ctx)
			assert.NoError(t, err)
		})
	}

	// Verify global instances are properly set
	assert.NotNil(t, GetGlobalMetrics())
	assert.NotNil(t, GetGlobalTracer())
	assert.NotNil(t, GetGlobalLogger())
}

// TestMetricsNilHandling tests that nil metrics don't cause panics
func TestMetricsNilHandling(t *testing.T) {
	ctx := context.Background()

	// Test with completely nil metrics
	var nilMetrics *Metrics
	nilMetrics.RecordOperation(ctx, "test", MemoryTypeBuffer, true)
	nilMetrics.RecordOperationDuration(ctx, "load", MemoryTypeBuffer, time.Second)
	nilMetrics.RecordError(ctx, "test", MemoryTypeBuffer, "error")
	nilMetrics.RecordMemorySize(ctx, MemoryTypeBuffer, 100)
	nilMetrics.RecordActiveMemory(ctx, MemoryTypeBuffer, 1)

	// Test with metrics having nil instruments
	mockMeter := NewMockMeterProvider().Meter()
	partialMetrics := &Metrics{meter: mockMeter}
	partialMetrics.RecordOperation(ctx, "test", MemoryTypeBuffer, true)
	partialMetrics.RecordOperationDuration(ctx, "load", MemoryTypeBuffer, time.Second)
	partialMetrics.RecordError(ctx, "test", MemoryTypeBuffer, "error")
	partialMetrics.RecordMemorySize(ctx, MemoryTypeBuffer, 100)
	partialMetrics.RecordActiveMemory(ctx, MemoryTypeBuffer, 1)
}

// TestTracerNilHandling tests that nil tracer doesn't cause panics
func TestTracerNilHandling(t *testing.T) {
	ctx := context.Background()

	// Test with nil tracer
	var nilTracer *Tracer
	ctx, span := nilTracer.StartSpan(ctx, "test", MemoryTypeBuffer, "test_memory")
	assert.Equal(t, ctx, context.Background()) // Should return original context
	assert.Nil(t, span)

	// Record error on nil span (should not panic)
	nilTracer.RecordSpanError(nil, errors.New("test error"))
}

// TestLoggerNilHandling tests that nil logger doesn't cause panics
func TestLoggerNilHandling(t *testing.T) {
	ctx := context.Background()

	// Test with nil logger
	var nilLogger *Logger
	nilLogger.LogMemoryOperation(ctx, slog.LevelInfo, "test", MemoryTypeBuffer, "test_memory", 1, time.Second, nil)
	nilLogger.LogMemoryLifecycle(ctx, "test", MemoryTypeBuffer, "test_memory")
	nilLogger.LogError(ctx, errors.New("test"), "test", MemoryTypeBuffer, "test_memory")
}

// TestGlobalFunctionsNilHandling tests that global functions handle nil gracefully
func TestGlobalFunctionsNilHandling(t *testing.T) {
	ctx := context.Background()

	// Reset global instances to nil
	globalMetrics = nil
	globalTracer = nil
	globalLogger = nil

	// These should not panic
	GetGlobalMetrics()
	GetGlobalTracer()
	GetGlobalLogger()

	// Convenience functions should not panic
	LogMemoryOperation(ctx, slog.LevelInfo, "test", MemoryTypeBuffer, "test", 1, time.Second, nil)
	LogMemoryLifecycle(ctx, "test", MemoryTypeBuffer, "test")
	LogError(ctx, errors.New("test"), "test", MemoryTypeBuffer, "test")
}

// BenchmarkMetricsRecording benchmarks metrics recording performance
func BenchmarkMetricsRecording(b *testing.B) {
	ctx := context.Background()
	mockMeter := NewMockMeterProvider().Meter()
	metrics := NewMetrics(mockMeter)

	b.Run("RecordOperation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			metrics.RecordOperation(ctx, "load", MemoryTypeBuffer, true)
		}
	})

	b.Run("RecordOperationDuration", func(b *testing.B) {
		duration := time.Millisecond * 100
		for i := 0; i < b.N; i++ {
			metrics.RecordOperationDuration(ctx, "save", MemoryTypeBuffer, duration)
		}
	})

	b.Run("RecordError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			metrics.RecordError(ctx, "load", MemoryTypeBuffer, "timeout")
		}
	})
}

// BenchmarkTracingOperations benchmarks tracing operations performance
func BenchmarkTracingOperations(b *testing.B) {
	ctx := context.Background()
	tracer := NewTracer()

	b.Run("StartSpan", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, span := tracer.StartSpan(ctx, "test", MemoryTypeBuffer, "test_memory")
			span.End()
		}
	})

	b.Run("RecordSpanError", func(b *testing.B) {
		testErr := errors.New("test error")
		for i := 0; i < b.N; i++ {
			_, span := tracer.StartSpan(ctx, "test", MemoryTypeBuffer, "test_memory")
			tracer.RecordSpanError(span, testErr)
			span.End()
		}
	})
}

// BenchmarkLoggingOperations benchmarks logging operations performance
func BenchmarkLoggingOperations(b *testing.B) {
	ctx := context.Background()
	logger := NewLogger(nil)

	duration := 50 * time.Millisecond
	testErr := errors.New("benchmark error")

	b.Run("LogMemoryOperation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.LogMemoryOperation(ctx, slog.LevelInfo, "load", MemoryTypeBuffer, "memory", 5, duration, nil)
		}
	})

	b.Run("LogMemoryOperationWithError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.LogMemoryOperation(ctx, slog.LevelError, "save", MemoryTypeBuffer, "memory", 3, duration, testErr)
		}
	})

	b.Run("LogMemoryLifecycle", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.LogMemoryLifecycle(ctx, "created", MemoryTypeBuffer, "memory")
		}
	})
}
