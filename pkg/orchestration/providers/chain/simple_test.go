package chain

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRunnable is a mock implementation of core.Runnable for testing
type MockRunnable struct {
	mock.Mock
	name string
}

func (m *MockRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0), args.Error(1)
}

func (m *MockRunnable) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	args := m.Called(ctx, inputs, opts)
	return args.Get(0).([]any), args.Error(1)
}

func (m *MockRunnable) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0).(<-chan any), args.Error(1)
}

// MockMemory is a mock implementation of memory.Memory for testing
type MockMemory struct {
	mock.Mock
}

func (m *MockMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	args := m.Called(ctx, inputs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]any), args.Error(1)
}

func (m *MockMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	args := m.Called(ctx, inputs, outputs)
	return args.Error(0)
}

func (m *MockMemory) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMemory) MemoryVariables() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

// MockTracer is a mock implementation of trace.Tracer for testing
type MockTracer struct {
	mock.Mock
}

func (m *MockTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	args := m.Called(ctx, spanName, opts)
	return args.Get(0).(context.Context), args.Get(1).(trace.Span)
}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewSimpleChain(t *testing.T) {
	config := iface.ChainConfig{
		Name:  "test-chain",
		Steps: []core.Runnable{&MockRunnable{name: "step1"}},
	}

	memory := &MockMemory{}

	chain := NewSimpleChain(config, memory, nil)

	assert.NotNil(t, chain)
	assert.Equal(t, config, chain.config)
	assert.Equal(t, memory, chain.memory)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestSimpleChain_GetInputKeys(t *testing.T) {
	tests := []struct {
		name         string
		inputKeys    []string
		expectedKeys []string
	}{
		{
			name:         "with custom input keys",
			inputKeys:    []string{"key1", "key2"},
			expectedKeys: []string{"key1", "key2"},
		},
		{
			name:         "with empty input keys",
			inputKeys:    []string{},
			expectedKeys: []string{"input"},
		},
		{
			name:         "with nil input keys",
			inputKeys:    nil,
			expectedKeys: []string{"input"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := iface.ChainConfig{InputKeys: tt.inputKeys}
			chain := NewSimpleChain(config, nil, nil)

			keys := chain.GetInputKeys()
			assert.Equal(t, tt.expectedKeys, keys)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		})
	}
}

func TestSimpleChain_GetOutputKeys(t *testing.T) {
	tests := []struct {
		name         string
		outputKeys   []string
		expectedKeys []string
	}{
		{
			name:         "with custom output keys",
			outputKeys:   []string{"result", "metadata"},
			expectedKeys: []string{"result", "metadata"},
		},
		{
			name:         "with empty output keys",
			outputKeys:   []string{},
			expectedKeys: []string{"output"},
		},
		{
			name:         "with nil output keys",
			outputKeys:   nil,
			expectedKeys: []string{"output"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := iface.ChainConfig{OutputKeys: tt.outputKeys}
			chain := NewSimpleChain(config, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			keys := chain.GetOutputKeys()
			assert.Equal(t, tt.expectedKeys, keys)
		})
	}
}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

func TestSimpleChain_GetMemory(t *testing.T) {
	memory := &MockMemory{}
	chain := NewSimpleChain(iface.ChainConfig{}, memory, nil)

	assert.Equal(t, memory, chain.GetMemory())
}

func TestSimpleChain_Invoke_Success(t *testing.T) {
	step1 := &MockRunnable{name: "step1"}
	step1.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"result": "success"}, nil)

	config := iface.ChainConfig{
		Name:  "test-chain",
		Steps: []core.Runnable{step1},
	}

	chain := NewSimpleChain(config, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	input := map[string]any{"input": "test"}
	result, err := chain.Invoke(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, map[string]any{"result": "success"}, result)

	step1.AssertExpectations(t)
}

func TestSimpleChain_Invoke_WithMemory(t *testing.T) {
	mockMemory := &MockMemory{}
	mockMemory.On("LoadMemoryVariables", mock.Anything, mock.AnythingOfType("map[string]interface {}")).Return(map[string]any{"memory": "data"}, nil)
	mockMemory.On("SaveContext", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	step1 := &MockRunnable{name: "step1"}
	step1.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"result": "with_memory"}, nil)

	config := iface.ChainConfig{
		Name:       "test-chain",
		Steps:      []core.Runnable{step1},
		InputKeys:  []string{"input"},
		OutputKeys: []string{"result"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	chain := NewSimpleChain(config, mockMemory, nil)

	input := map[string]any{"input": "test"}
	result, err := chain.Invoke(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	step1.AssertExpectations(t)
	mockMemory.AssertExpectations(t)
}

func TestSimpleChain_Invoke_WithTimeout(t *testing.T) {
	step1 := &MockRunnable{name: "step1"}
	step1.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"result": "success"}, nil)

	config := iface.ChainConfig{
		Name:    "test-chain",
		Steps:   []core.Runnable{step1},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		Timeout: 5, // 5 seconds
	}

	chain := NewSimpleChain(config, nil, nil)

	input := map[string]any{"input": "test"}
	result, err := chain.Invoke(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	step1.AssertExpectations(t)
}

func TestSimpleChain_Invoke_StepError(t *testing.T) {
	step1 := &MockRunnable{name: "step1"}
	step1.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(nil, errors.New("step failed"))

	config := iface.ChainConfig{
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		Name:  "test-chain",
		Steps: []core.Runnable{step1},
	}

	chain := NewSimpleChain(config, nil, nil)

	input := map[string]any{"input": "test"}
	result, err := chain.Invoke(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "error in chain step")

	step1.AssertExpectations(t)
}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestSimpleChain_Invoke_InvalidInput(t *testing.T) {
	mockMemory := &MockMemory{}

	config := iface.ChainConfig{
		Name:  "test-chain",
		Steps: []core.Runnable{&MockRunnable{name: "step1"}},
	}

	chain := NewSimpleChain(config, mockMemory, nil)

	// Test with invalid input type
	result, err := chain.Invoke(context.Background(), 123)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "chain input must be map[string]any")
}

func TestSimpleChain_Invoke_MemoryError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	mockMemory := &MockMemory{}
	mockMemory.On("LoadMemoryVariables", mock.Anything, mock.AnythingOfType("map[string]interface {}")).Return(nil, errors.New("memory load failed"))

	config := iface.ChainConfig{
		Name:  "test-chain",
		Steps: []core.Runnable{&MockRunnable{name: "step1"}},
	}

	chain := NewSimpleChain(config, mockMemory, nil)

	input := map[string]any{"input": "test"}
	result, err := chain.Invoke(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "memory load failed")

	mockMemory.AssertExpectations(t)
}

func TestSimpleChain_Batch(t *testing.T) {
	step1 := &MockRunnable{name: "step1"}
	step1.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"result": "success"}, nil).Maybe()

	config := iface.ChainConfig{
		Name:  "test-chain",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		Steps: []core.Runnable{step1},
	}

	chain := NewSimpleChain(config, nil, nil)

	inputs := []any{
		map[string]any{"input": "test1"},
		map[string]any{"input": "test2"},
		map[string]any{"input": "test3"},
	}

	results, err := chain.Batch(context.Background(), inputs)

	assert.NoError(t, err)
	assert.Len(t, results, 3)
	for _, result := range results {
		assert.NotNil(t, result)
	}

	step1.AssertExpectations(t)
}

func TestSimpleChain_Batch_WithErrors(t *testing.T) {
	step1 := &MockRunnable{name: "step1"}
	step1.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).
		Return(nil, errors.New("step failed")).Maybe()
	step1.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).
		Return(map[string]any{"result": "success"}, nil).Maybe()

	config := iface.ChainConfig{
		Name:  "test-chain",
		Steps: []core.Runnable{step1},
	}

	chain := NewSimpleChain(config, nil, nil)

	inputs := []any{
		map[string]any{"input": "test1"},
		map[string]any{"input": "test2"},
		map[string]any{"input": "test3"},
	}

	results, err := chain.Batch(context.Background(), inputs)

	assert.Error(t, err)
	assert.Len(t, results, 3)
	assert.Contains(t, err.Error(), "error processing batch item")

	step1.AssertExpectations(t)
}

func TestSimpleChain_Stream_Success(t *testing.T) {
	step1 := &MockRunnable{name: "step1"}
	step1.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(map[string]any{"intermediate": "result"}, nil)

	step2 := &MockRunnable{name: "step2"}
	streamChan := make(chan any, 1)
	streamChan <- "stream_result"
	close(streamChan)
	step2.On("Stream", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return((<-chan any)(streamChan), nil)

	config := iface.ChainConfig{
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		Name:  "test-chain",
		Steps: []core.Runnable{step1, step2},
	}

	chain := NewSimpleChain(config, nil, nil)

	input := map[string]any{"input": "test"}
	resultChan, err := chain.Stream(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, resultChan)

	// Read from the stream
	select {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	case result := <-resultChan:
		assert.Equal(t, "stream_result", result)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected result from stream")
	}

	step1.AssertExpectations(t)
	step2.AssertExpectations(t)
}

func TestSimpleChain_Stream_EmptyChain(t *testing.T) {
	config := iface.ChainConfig{
		Name:  "empty-chain",
		Steps: []core.Runnable{},
	}

	chain := NewSimpleChain(config, nil, nil)

	input := map[string]any{"input": "test"}
	resultChan, err := chain.Stream(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, resultChan)
	assert.Contains(t, err.Error(), "cannot stream an empty chain")
}

func TestSimpleChain_Stream_PrecomputeError(t *testing.T) {
	step1 := &MockRunnable{name: "step1"}
	step1.On("Invoke", mock.Anything, mock.AnythingOfType("map[string]interface {}"), mock.Anything).Return(nil, errors.New("precompute failed"))

	step2 := &MockRunnable{name: "step2"}

	config := iface.ChainConfig{
		Name:  "test-chain",
		Steps: []core.Runnable{step1, step2},
	}

	chain := NewSimpleChain(config, nil, nil)

	input := map[string]any{"input": "test"}
	resultChan, err := chain.Stream(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, resultChan)
	assert.Contains(t, err.Error(), "error in chain stream pre-computation")

	step1.AssertExpectations(t)
}

// MockSpan is a mock implementation of trace.Span
type MockSpan struct {
	mock.Mock
}

func (m *MockSpan) End(options ...trace.SpanEndOption) {
	m.Called(options)
}

func (m *MockSpan) AddEvent(name string, options ...trace.EventOption) {
	m.Called(name, options)
}

func (m *MockSpan) IsRecording() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSpan) RecordError(err error, options ...trace.EventOption) {
	m.Called(err, options)
}

func (m *MockSpan) SpanContext() trace.SpanContext {
	args := m.Called()
	return args.Get(0).(trace.SpanContext)
}

func (m *MockSpan) SetStatus(code codes.Code, description string) {
	m.Called(code, description)
}

func (m *MockSpan) SetName(name string) {
	m.Called(name)
}

func (m *MockSpan) SetAttributes(kv ...attribute.KeyValue) {
	m.Called(kv)
}

func (m *MockSpan) TracerProvider() trace.TracerProvider {
	args := m.Called()
	return args.Get(0).(trace.TracerProvider)
}
