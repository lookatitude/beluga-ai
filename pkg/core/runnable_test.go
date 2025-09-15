package core

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

// MockRunnable is a test implementation of the Runnable interface
type MockRunnable struct {
	invokeFunc func(ctx context.Context, input any, options ...Option) (any, error)
	batchFunc  func(ctx context.Context, inputs []any, options ...Option) ([]any, error)
	streamFunc func(ctx context.Context, input any, options ...Option) (<-chan any, error)
}

func (m *MockRunnable) Invoke(ctx context.Context, input any, options ...Option) (any, error) {
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, input, options...)
	}
	return "mock_result", nil
}

func (m *MockRunnable) Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error) {
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
	if m.streamFunc != nil {
		return m.streamFunc(ctx, input, options...)
	}
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		ch <- "mock_stream_result"
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
	mock := &MockRunnable{
		invokeFunc: func(ctx context.Context, input any, options ...Option) (any, error) {
			return "custom_result", nil
		},
	}

	result, err := mock.Invoke(context.Background(), "test_input")
	if err != nil {
		t.Errorf("MockRunnable.Invoke() error = %v", err)
		return
	}

	if result != "custom_result" {
		t.Errorf("MockRunnable.Invoke() = %v, expected custom_result", result)
	}
}

func TestMockRunnable_Batch(t *testing.T) {
	mock := &MockRunnable{}

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
}

func TestTracedRunnable_Invoke(t *testing.T) {
	mock := &MockRunnable{
		invokeFunc: func(ctx context.Context, input any, options ...Option) (any, error) {
			return "traced_result", nil
		},
	}

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
}

func TestTracedRunnable_InvokeWithError(t *testing.T) {
	expectedErr := errors.New("test error")
	mock := &MockRunnable{
		invokeFunc: func(ctx context.Context, input any, options ...Option) (any, error) {
			return nil, expectedErr
		},
	}

	tracer := trace.NewNoopTracerProvider().Tracer("")
	metrics := NoOpMetrics()
	traced := NewTracedRunnable(mock, tracer, metrics, "test_component", "test_name")

	_, err := traced.Invoke(context.Background(), "test_input")
	if err != expectedErr {
		t.Errorf("TracedRunnable.Invoke() error = %v, expected %v", err, expectedErr)
	}
}

func TestTracedRunnable_Batch(t *testing.T) {
	mock := &MockRunnable{
		batchFunc: func(ctx context.Context, inputs []any, options ...Option) ([]any, error) {
			results := make([]any, len(inputs))
			for i := range inputs {
				results[i] = "traced_batch_result"
			}
			return results, nil
		},
	}

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
		if result != "traced_batch_result" {
			t.Errorf("TracedRunnable.Batch() result[%d] = %v, expected traced_batch_result", i, result)
		}
	}
}

func TestTracedRunnable_Stream(t *testing.T) {
	mock := &MockRunnable{
		streamFunc: func(ctx context.Context, input any, options ...Option) (<-chan any, error) {
			ch := make(chan any, 2)
			go func() {
				defer close(ch)
				ch <- "chunk1"
				ch <- "chunk2"
			}()
			return ch, nil
		},
	}

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
}

func TestTracedRunnable_StreamWithError(t *testing.T) {
	expectedErr := errors.New("stream error")
	mock := &MockRunnable{
		streamFunc: func(ctx context.Context, input any, options ...Option) (<-chan any, error) {
			return nil, expectedErr
		},
	}

	tracer := trace.NewNoopTracerProvider().Tracer("")
	metrics := NoOpMetrics()
	traced := NewTracedRunnable(mock, tracer, metrics, "test_component", "test_name")

	_, err := traced.Stream(context.Background(), "test_input")
	if err != expectedErr {
		t.Errorf("TracedRunnable.Stream() error = %v, expected %v", err, expectedErr)
	}
}

func TestRunnableWithTracing(t *testing.T) {
	mock := &MockRunnable{}

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
	mock := &MockRunnable{}

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

func TestBuilder_RegisterMonitoringComponents(t *testing.T) {
	builder := NewBuilder(NewContainer())

	// Test registering no-op monitoring components
	if err := builder.RegisterNoOpLogger(); err != nil {
		t.Errorf("RegisterNoOpLogger() error = %v", err)
	}

	if err := builder.RegisterNoOpTracerProvider(); err != nil {
		t.Errorf("RegisterNoOpTracerProvider() error = %v", err)
	}

	if err := builder.RegisterNoOpMetrics(); err != nil {
		t.Errorf("RegisterNoOpMetrics() error = %v", err)
	}

	// Test resolving the components
	var logger Logger
	if err := builder.Build(&logger); err != nil {
		t.Errorf("Build(logger) error = %v", err)
	}
	if logger == nil {
		t.Error("Expected logger to be resolved, got nil")
	}

	var tracerProvider TracerProvider
	if err := builder.Build(&tracerProvider); err != nil {
		t.Errorf("Build(tracerProvider) error = %v", err)
	}
	if tracerProvider == nil {
		t.Error("Expected tracerProvider to be resolved, got nil")
	}

	var metrics *Metrics
	if err := builder.Build(&metrics); err != nil {
		t.Errorf("Build(metrics) error = %v", err)
	}
	if metrics == nil {
		t.Error("Expected metrics to be resolved, got nil")
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
			if err.Cause != cause {
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

