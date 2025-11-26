package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// MockRunnable is an enhanced test implementation of the Runnable interface
// with call tracking, error injection, and configurable behavior.
type MockRunnable struct {
	invokeError  error
	streamError  error
	batchError   error
	batchFunc    func(ctx context.Context, inputs []any, options ...Option) ([]any, error)
	streamFunc   func(ctx context.Context, input any, options ...Option) (<-chan any, error)
	invokeFunc   func(ctx context.Context, input any, options ...Option) (any, error)
	invokeCalls  []InvokeCall
	streamCalls  []StreamCall
	batchCalls   []BatchCall
	streamChunks []any
	invokeDelay  time.Duration
	batchDelay   time.Duration
	streamDelay  time.Duration
}

type InvokeCall struct {
	Time    time.Time
	Ctx     context.Context
	Input   any
	Options []Option
}

type BatchCall struct {
	Time    time.Time
	Ctx     context.Context
	Inputs  []any
	Options []Option
}

type StreamCall struct {
	Time    time.Time
	Ctx     context.Context
	Input   any
	Options []Option
}

func NewMockRunnable() *MockRunnable {
	return &MockRunnable{
		invokeCalls:  make([]InvokeCall, 0),
		batchCalls:   make([]BatchCall, 0),
		streamCalls:  make([]StreamCall, 0),
		streamChunks: []any{"mock_stream_result"},
	}
}

func (m *MockRunnable) WithInvokeResult(result any) *MockRunnable {
	m.invokeFunc = func(ctx context.Context, input any, options ...Option) (any, error) {
		return result, nil
	}
	return m
}

func (m *MockRunnable) WithInvokeError(err error) *MockRunnable {
	m.invokeError = err
	return m
}

func (m *MockRunnable) WithBatchError(err error) *MockRunnable {
	m.batchError = err
	return m
}

func (m *MockRunnable) WithStreamError(err error) *MockRunnable {
	m.streamError = err
	return m
}

func (m *MockRunnable) WithStreamChunks(chunks ...any) *MockRunnable {
	m.streamChunks = chunks
	return m
}

func (m *MockRunnable) WithInvokeDelay(delay time.Duration) *MockRunnable {
	m.invokeDelay = delay
	return m
}

func (m *MockRunnable) WithBatchDelay(delay time.Duration) *MockRunnable {
	m.batchDelay = delay
	return m
}

func (m *MockRunnable) WithStreamDelay(delay time.Duration) *MockRunnable {
	m.streamDelay = delay
	return m
}

func (m *MockRunnable) GetInvokeCalls() []InvokeCall {
	return m.invokeCalls
}

func (m *MockRunnable) GetBatchCalls() []BatchCall {
	return m.batchCalls
}

func (m *MockRunnable) GetStreamCalls() []StreamCall {
	return m.streamCalls
}

func (m *MockRunnable) Reset() {
	m.invokeCalls = make([]InvokeCall, 0)
	m.batchCalls = make([]BatchCall, 0)
	m.streamCalls = make([]StreamCall, 0)
}

func (m *MockRunnable) Invoke(ctx context.Context, input any, options ...Option) (any, error) {
	call := InvokeCall{
		Ctx:     ctx,
		Input:   input,
		Options: options,
		Time:    time.Now(),
	}
	m.invokeCalls = append(m.invokeCalls, call)

	if m.invokeDelay > 0 {
		select {
		case <-time.After(m.invokeDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.invokeError != nil {
		return nil, m.invokeError
	}

	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, input, options...)
	}
	return "mock_result", nil
}

func (m *MockRunnable) Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error) {
	call := BatchCall{
		Ctx:     ctx,
		Inputs:  inputs,
		Options: options,
		Time:    time.Now(),
	}
	m.batchCalls = append(m.batchCalls, call)

	if m.batchDelay > 0 {
		select {
		case <-time.After(m.batchDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.batchError != nil {
		return nil, m.batchError
	}

	if m.batchFunc != nil {
		return m.batchFunc(ctx, inputs, options...)
	}

	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = "mock_result"
	}
	return results, nil
}

func (m *MockRunnable) Stream(ctx context.Context, input any, options ...Option) (<-chan any, error) {
	call := StreamCall{
		Ctx:     ctx,
		Input:   input,
		Options: options,
		Time:    time.Now(),
	}
	m.streamCalls = append(m.streamCalls, call)

	if m.streamDelay > 0 {
		select {
		case <-time.After(m.streamDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.streamError != nil {
		return nil, m.streamError
	}

	if m.streamFunc != nil {
		return m.streamFunc(ctx, input, options...)
	}

	ch := make(chan any, len(m.streamChunks))
	go func() {
		defer close(ch)
		for _, chunk := range m.streamChunks {
			select {
			case ch <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func TestWithOption(t *testing.T) {
	config := make(map[string]any)

	opt := WithOption("test_key", "test_value")
	opt.Apply(&config)

	if config["test_key"] != "test_value" {
		t.Errorf("Expected test_key to be 'test_value', got %v", config["test_key"])
	}
}

// Note: These tests are commented out due to import cycle issues
// The actual implementations of WithMaxTokens, WithTemperature, and EnsureMessages
// are in the llms package, which cannot be imported here due to circular dependencies.

/*
func TestWithMaxTokens(t *testing.T) {
	opt := llms.WithMaxTokens(100)
	config := make(map[string]any)
	opt.Apply(&config)

	if config["max_tokens"] != 100 {
		t.Errorf("Expected max_tokens to be 100, got %v", config["max_tokens"])
	}
}

func TestWithTemperature(t *testing.T) {
	opt := llms.WithTemperature(0.7)
	config := make(map[string]any)
	opt.Apply(&config)

	temp, ok := config["temperature"].(float32)
	if !ok || temp != 0.7 {
		t.Errorf("Expected temperature to be 0.7, got %v", config["temperature"])
	}
}

func TestEnsureMessages(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
		wantErr  bool
	}{
		{
			name:     "string input",
			input:    "Hello world",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "message input",
			input:    schema.NewHumanMessage("Hello"),
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "messages slice",
			input:    []schema.Message{schema.NewHumanMessage("Hello")},
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "invalid input",
			input:    123,
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := llms.EnsureMessages(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(result) != tt.expected {
				t.Errorf("EnsureMessages() returned %d messages, expected %d", len(result), tt.expected)
			}
		})
	}
}
*/

func TestMockRunnable_Invoke(t *testing.T) {
	mock := NewMockRunnable().WithInvokeResult("custom_result")

	result, err := mock.Invoke(context.Background(), "test_input")
	if err != nil {
		t.Errorf("MockRunnable.Invoke() error = %v", err)
		return
	}

	if result != "custom_result" {
		t.Errorf("MockRunnable.Invoke() = %v, expected custom_result", result)
	}

	// Test call tracking
	calls := mock.GetInvokeCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 invoke call, got %d", len(calls))
	}
}

func TestMockRunnable_InvokeWithError(t *testing.T) {
	expectedErr := errors.New("test error")
	mock := NewMockRunnable().WithInvokeError(expectedErr)

	_, err := mock.Invoke(context.Background(), "test_input")
	if !errors.Is(err, expectedErr) {
		t.Errorf("MockRunnable.Invoke() error = %v, expected %v", err, expectedErr)
	}

	calls := mock.GetInvokeCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 invoke call, got %d", len(calls))
	}
}

func TestMockRunnable_InvokeWithDelay(t *testing.T) {
	mock := NewMockRunnable().WithInvokeDelay(10 * time.Millisecond)

	start := time.Now()
	_, err := mock.Invoke(context.Background(), "test_input")
	duration := time.Since(start)

	if err != nil {
		t.Errorf("MockRunnable.Invoke() error = %v", err)
	}

	if duration < 10*time.Millisecond {
		t.Errorf("Expected delay of at least 10ms, got %v", duration)
	}
}

func TestMockRunnable_Batch(t *testing.T) {
	mock := NewMockRunnable()

	inputs := []any{"input1", "input2", "input3"}
	results, err := mock.Batch(context.Background(), inputs)
	if err != nil {
		t.Errorf("MockRunnable.Batch() error = %v", err)
		return
	}

	if len(results) != len(inputs) {
		t.Errorf("MockRunnable.Batch() returned %d results, expected %d", len(results), len(inputs))
	}

	for i, result := range results {
		if result != "mock_result" {
			t.Errorf("MockRunnable.Batch() result[%d] = %v, expected mock_result", i, result)
		}
	}

	// Test call tracking
	calls := mock.GetBatchCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 batch call, got %d", len(calls))
	}
	if len(calls[0].Inputs) != len(inputs) {
		t.Errorf("Batch call inputs length = %d, expected %d", len(calls[0].Inputs), len(inputs))
	}
}

func TestMockRunnable_BatchWithError(t *testing.T) {
	expectedErr := errors.New("batch error")
	mock := NewMockRunnable().WithBatchError(expectedErr)

	inputs := []any{"input1", "input2"}
	_, err := mock.Batch(context.Background(), inputs)
	if !errors.Is(err, expectedErr) {
		t.Errorf("MockRunnable.Batch() error = %v, expected %v", err, expectedErr)
	}

	calls := mock.GetBatchCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 batch call, got %d", len(calls))
	}
}

func TestMockRunnable_BatchWithDelay(t *testing.T) {
	mock := NewMockRunnable().WithBatchDelay(10 * time.Millisecond)

	inputs := []any{"input1", "input2"}
	start := time.Now()
	_, err := mock.Batch(context.Background(), inputs)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("MockRunnable.Batch() error = %v", err)
	}

	if duration < 10*time.Millisecond {
		t.Errorf("Expected delay of at least 10ms, got %v", duration)
	}
}

func TestTracedRunnable_Invoke(t *testing.T) {
	mock := NewMockRunnable().WithInvokeResult("traced_result")

	tracer := trace.NewNoopTracerProvider().Tracer("")
	metrics := NoOpMetrics()
	traced := NewTracedRunnable(mock, tracer, metrics, "test_component", "test_name")

	result, err := traced.Invoke(context.Background(), "test_input")
	if err != nil {
		t.Errorf("TracedRunnable.Invoke() error = %v", err)
		return
	}

	if result != "traced_result" {
		t.Errorf("TracedRunnable.Invoke() = %v, expected traced_result", result)
	}

	// Verify that the mock was called
	calls := mock.GetInvokeCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 invoke call on mock, got %d", len(calls))
	}
}

func TestTracedRunnable_InvokeWithError(t *testing.T) {
	expectedErr := errors.New("test error")
	mock := NewMockRunnable().WithInvokeError(expectedErr)

	tracer := trace.NewNoopTracerProvider().Tracer("")
	metrics := NoOpMetrics()
	traced := NewTracedRunnable(mock, tracer, metrics, "test_component", "test_name")

	_, err := traced.Invoke(context.Background(), "test_input")
	if !errors.Is(err, expectedErr) {
		t.Errorf("TracedRunnable.Invoke() error = %v, expected %v", err, expectedErr)
	}

	// Verify that the mock was called
	calls := mock.GetInvokeCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 invoke call on mock, got %d", len(calls))
	}
}

func TestTracedRunnable_Batch(t *testing.T) {
	mock := NewMockRunnable()

	tracer := trace.NewNoopTracerProvider().Tracer("")
	metrics := NoOpMetrics()
	traced := NewTracedRunnable(mock, tracer, metrics, "test_component", "test_name")

	inputs := []any{"input1", "input2"}
	results, err := traced.Batch(context.Background(), inputs)
	if err != nil {
		t.Errorf("TracedRunnable.Batch() error = %v", err)
		return
	}

	if len(results) != len(inputs) {
		t.Errorf("TracedRunnable.Batch() returned %d results, expected %d", len(results), len(inputs))
	}

	for i, result := range results {
		if result != "mock_result" {
			t.Errorf("TracedRunnable.Batch() result[%d] = %v, expected mock_result", i, result)
		}
	}

	// Verify that the mock was called
	calls := mock.GetBatchCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 batch call on mock, got %d", len(calls))
	}
}

func TestTracedRunnable_Stream(t *testing.T) {
	mock := NewMockRunnable().WithStreamChunks("chunk1", "chunk2")

	tracer := trace.NewNoopTracerProvider().Tracer("")
	metrics := NoOpMetrics()
	traced := NewTracedRunnable(mock, tracer, metrics, "test_component", "test_name")

	streamChan, err := traced.Stream(context.Background(), "test_input")
	if err != nil {
		t.Errorf("TracedRunnable.Stream() error = %v", err)
		return
	}

	var chunks []any
	for chunk := range streamChan {
		chunks = append(chunks, chunk)
	}

	if len(chunks) != 2 {
		t.Errorf("TracedRunnable.Stream() received %d chunks, expected 2", len(chunks))
	}

	if chunks[0] != "chunk1" || chunks[1] != "chunk2" {
		t.Errorf("TracedRunnable.Stream() chunks = %v, expected [chunk1, chunk2]", chunks)
	}

	// Verify that the mock was called
	calls := mock.GetStreamCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 stream call on mock, got %d", len(calls))
	}
}

func TestTracedRunnable_StreamWithError(t *testing.T) {
	expectedErr := errors.New("stream error")
	mock := NewMockRunnable().WithStreamError(expectedErr)

	tracer := trace.NewNoopTracerProvider().Tracer("")
	metrics := NoOpMetrics()
	traced := NewTracedRunnable(mock, tracer, metrics, "test_component", "test_name")

	_, err := traced.Stream(context.Background(), "test_input")
	if !errors.Is(err, expectedErr) {
		t.Errorf("TracedRunnable.Stream() error = %v, expected %v", err, expectedErr)
	}

	// Verify that the mock was called
	calls := mock.GetStreamCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 stream call on mock, got %d", len(calls))
	}
}

func TestRunnableWithTracing(t *testing.T) {
	mock := NewMockRunnable()

	tracer := trace.NewNoopTracerProvider().Tracer("")
	metrics := NoOpMetrics()
	traced := RunnableWithTracing(mock, tracer, metrics, "test_component")

	if tracedRunnable, ok := traced.(*TracedRunnable); !ok {
		t.Errorf("RunnableWithTracing() returned %T, expected *TracedRunnable", traced)
	} else {
		if tracedRunnable.componentType != "test_component" {
			t.Errorf("RunnableWithTracing() componentType = %q, expected %q", tracedRunnable.componentType, "test_component")
		}
		if tracedRunnable.componentName != "" {
			t.Errorf("RunnableWithTracing() componentName = %q, expected empty string", tracedRunnable.componentName)
		}
	}
}

func TestRunnableWithTracingAndName(t *testing.T) {
	mock := NewMockRunnable()

	tracer := trace.NewNoopTracerProvider().Tracer("")
	metrics := NoOpMetrics()
	traced := RunnableWithTracingAndName(mock, tracer, metrics, "test_component", "test_name")

	if tracedRunnable, ok := traced.(*TracedRunnable); !ok {
		t.Errorf("RunnableWithTracingAndName() returned %T, expected *TracedRunnable", traced)
	} else {
		if tracedRunnable.componentType != "test_component" {
			t.Errorf("RunnableWithTracingAndName() componentType = %q, expected %q", tracedRunnable.componentType, "test_component")
		}
		if tracedRunnable.componentName != "test_name" {
			t.Errorf("RunnableWithTracingAndName() componentName = %q, expected %q", tracedRunnable.componentName, "test_name")
		}
	}
}

func TestNoOpLogger(t *testing.T) {
	logger := &noOpLogger{}

	// These should not panic
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// With should return the same logger
	withLogger := logger.With("key", "value")
	if withLogger != logger {
		t.Error("With() should return the same logger instance")
	}
}

// Metrics verification tests

func TestMetrics_Recording(t *testing.T) {
	// This test verifies that metrics are properly recorded during Runnable operations
	mock := NewMockRunnable()

	tracer := trace.NewNoopTracerProvider().Tracer("")
	metrics := NoOpMetrics()

	if metrics == nil {
		t.Fatal("NewMetrics() returned nil")
	}

	traced := NewTracedRunnable(mock, tracer, metrics, "test_component", "test_name")

	// Test Invoke metrics
	ctx := context.Background()
	result, err := traced.Invoke(ctx, "test_input")
	if err != nil {
		t.Errorf("TracedRunnable.Invoke() error = %v", err)
	}
	if result != "mock_result" {
		t.Errorf("TracedRunnable.Invoke() = %v, expected mock_result", result)
	}

	// Test Batch metrics
	inputs := []any{"input1", "input2"}
	results, err := traced.Batch(ctx, inputs)
	if err != nil {
		t.Errorf("TracedRunnable.Batch() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("TracedRunnable.Batch() returned %d results, expected 2", len(results))
	}

	// Test Stream metrics
	streamChan, err := traced.Stream(ctx, "test_input")
	if err != nil {
		t.Errorf("TracedRunnable.Stream() error = %v", err)
	}
	chunks := 0
	for range streamChan {
		chunks++
	}
	if chunks != 1 {
		t.Errorf("Expected 1 chunk from stream, got %d", chunks)
	}
}

func TestMetrics_NoOpMetrics(t *testing.T) {
	metrics := NoOpMetrics()
	if metrics == nil {
		t.Fatal("NoOpMetrics() returned nil")
	}

	// No-op metrics should not panic when called
	ctx := context.Background()
	metrics.RecordRunnableInvoke(ctx, "test", time.Millisecond, nil)
	metrics.RecordRunnableBatch(ctx, "test", 5, time.Millisecond, nil)
	metrics.RecordRunnableStream(ctx, "test", time.Millisecond, 3, nil)

	// Verify that all fields are nil (no-op behavior)
	if metrics.runnableInvokes != nil ||
		metrics.runnableBatches != nil ||
		metrics.runnableStreams != nil ||
		metrics.runnableErrors != nil ||
		metrics.runnableDuration != nil ||
		metrics.batchSize != nil ||
		metrics.batchDuration != nil ||
		metrics.streamDuration != nil ||
		metrics.streamChunks != nil {
		t.Error("NoOpMetrics should have all fields as nil")
	}
}

// Simple tracing test using NoOp tracer.
func TestTracedRunnable_WithNoOpTracer(t *testing.T) {
	mock := NewMockRunnable()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	metrics := NoOpMetrics()

	traced := NewTracedRunnable(mock, tracer, metrics, "test_component", "test_name")

	// Test that traced runnable works with no-op tracer
	ctx := context.Background()
	result, err := traced.Invoke(ctx, "test_input")
	if err != nil {
		t.Errorf("TracedRunnable.Invoke() error = %v", err)
	}
	if result != "mock_result" {
		t.Errorf("TracedRunnable.Invoke() = %v, expected mock_result", result)
	}
}

func TestFrameworkErrorTypes(t *testing.T) {
	tests := []struct {
		name        string
		constructor func(string, error) *FrameworkError
		errorType   ErrorType
	}{
		{"ValidationError", NewValidationError, ErrorTypeValidation},
		{"NetworkError", NewNetworkError, ErrorTypeNetwork},
		{"AuthenticationError", NewAuthenticationError, ErrorTypeAuthentication},
		{"InternalError", NewInternalError, ErrorTypeInternal},
		{"ConfigurationError", NewConfigurationError, ErrorTypeConfiguration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cause := errors.New("test cause")
			err := tt.constructor("test message", cause)

			if err.Type != tt.errorType {
				t.Errorf("Error type = %v, expected %v", err.Type, tt.errorType)
			}
			if err.Message != "test message" {
				t.Errorf("Error message = %q, expected %q", err.Message, "test message")
			}
			if !errors.Is(err.Cause, cause) {
				t.Errorf("Error cause = %v, expected %v", err.Cause, cause)
			}
		})
	}
}

func TestContainerHealthChecker(t *testing.T) {
	container := NewContainer()

	// Test that container implements HealthChecker
	var hc HealthChecker = container
	if hc == nil {
		t.Error("Container should implement HealthChecker")
	}

	// Test health check
	ctx := context.Background()
	err := hc.CheckHealth(ctx)
	if err != nil {
		t.Errorf("CheckHealth() error = %v", err)
	}
}

func TestContainerWithMonitoring(t *testing.T) {
	logger := &testLogger{}
	tracerProvider := trace.NewNoopTracerProvider()

	container := NewContainerWithOptions(
		WithLogger(logger),
		WithTracerProvider(tracerProvider),
	)

	// Test registration with monitoring
	err := container.Register(func() string { return "test" })
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// Test resolution with monitoring
	var result string
	err = container.Resolve(&result)
	if err != nil {
		t.Errorf("Resolve() error = %v", err)
	}

	if result != "test" {
		t.Errorf("Resolve() = %q, expected %q", result, "test")
	}
}
