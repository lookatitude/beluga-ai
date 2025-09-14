package core

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockRunnable is a test implementation of the Runnable interface
type MockRunnable struct {
	invokeFunc  func(ctx context.Context, input any, options ...Option) (any, error)
	batchFunc   func(ctx context.Context, inputs []any, options ...Option) ([]any, error)
	streamFunc  func(ctx context.Context, input any, options ...Option) (<-chan any, error)
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
