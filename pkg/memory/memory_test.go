package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMemory is a mock implementation of the Memory interface for testing.
type MockMemory struct {
	MockMemoryVariables     func() []string
	MockLoadMemoryVariables func(ctx context.Context, inputs map[string]any) (map[string]any, error)
	MockSaveContext         func(ctx context.Context, inputs map[string]any, outputs map[string]any) error
	MockClear               func(ctx context.Context) error
}

func (m *MockMemory) MemoryVariables() []string {
	if m.MockMemoryVariables != nil {
		return m.MockMemoryVariables()
	}
	return []string{"mock_memory"}
}

func (m *MockMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	if m.MockLoadMemoryVariables != nil {
		return m.MockLoadMemoryVariables(ctx, inputs)
	}
	return map[string]any{"mock_loaded_var": "mock_value"}, nil
}

func (m *MockMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	if m.MockSaveContext != nil {
		return m.MockSaveContext(ctx, inputs, outputs)
	}
	return nil
}

func (m *MockMemory) Clear(ctx context.Context) error {
	if m.MockClear != nil {
		return m.MockClear(ctx)
	}
	return nil
}

// TestMemoryInterface ensures that MockMemory correctly implements the Memory interface.
func TestMemoryInterface(t *testing.T) {
	var _ Memory = (*MockMemory)(nil)

	// Example of how to use the mock for a basic test
	mockMem := &MockMemory{}
	ctx := context.Background()

	// Test LoadMemoryVariables
	loadedVars, err := mockMem.LoadMemoryVariables(ctx, nil)
	assert.NoError(t, err)
	assert.NotNil(t, loadedVars)
	assert.Equal(t, "mock_value", loadedVars["mock_loaded_var"])

	// Test SaveContext
	err = mockMem.SaveContext(ctx, nil, nil)
	assert.NoError(t, err)

	// Test Clear
	err = mockMem.Clear(context.Background())
	assert.NoError(t, err)
}

// TestFactory tests the memory factory functionality.
func TestFactory(t *testing.T) {
	ctx := context.Background()
	factory := NewFactory()

	// Test creating buffer memory
	config := Config{
		Type:           MemoryTypeBuffer,
		MemoryKey:      "test_history",
		ReturnMessages: false,
		Enabled:        true,
	}

	memory, err := factory.CreateMemory(ctx, config)
	require.NoError(t, err)
	assert.NotNil(t, memory)
	assert.Equal(t, []string{"test_history"}, memory.MemoryVariables())

	// Test loading memory variables
	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)
	assert.Contains(t, vars, "test_history")
}

// TestConvenienceFunctions tests the convenience functions for creating memory instances.
func TestConvenienceFunctions(t *testing.T) {
	ctx := context.Background()

	// Test buffer memory creation
	history := NewBaseChatMessageHistory()
	bufferMem := NewChatMessageBufferMemory(history)
	assert.NotNil(t, bufferMem)
	assert.Equal(t, []string{"history"}, bufferMem.MemoryVariables())

	// Test window memory creation
	windowMem := NewConversationBufferWindowMemory(history, 5, "window_history", false)
	assert.NotNil(t, windowMem)
	assert.Equal(t, []string{"window_history"}, windowMem.MemoryVariables())

	// Test saving and loading context
	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there"}

	err := bufferMem.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)

	vars, err := bufferMem.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)
	assert.Contains(t, vars, "history")
	assert.NotEmpty(t, vars["history"])
}

// TestConfiguration tests the configuration structs and options.
func TestConfiguration(t *testing.T) {
	// Test default configuration
	config := Config{
		Type:           MemoryTypeBuffer,
		MemoryKey:      "history",
		InputKey:       "input",
		OutputKey:      "output",
		ReturnMessages: false,
		WindowSize:     5,
		MaxTokenLimit:  2000,
		TopK:           4,
		HumanPrefix:    "Human",
		AIPrefix:       "AI",
		Enabled:        true,
	}

	// Test functional options
	WithMemoryKey("custom_history")(&config)
	assert.Equal(t, "custom_history", config.MemoryKey)

	WithWindowSize(10)(&config)
	assert.Equal(t, 10, config.WindowSize)

	WithReturnMessages(true)(&config)
	assert.Equal(t, true, config.ReturnMessages)
}

// TestErrorHandling tests error handling functionality.
func TestErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Test invalid configuration
	factory := NewFactory()
	invalidConfig := Config{
		Type:    "invalid_type",
		Enabled: true,
	}

	_, err := factory.CreateMemory(ctx, invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid memory configuration")

	// Test disabled memory
	disabledConfig := Config{
		Type:    MemoryTypeBuffer,
		Enabled: false,
	}

	memory, err := factory.CreateMemory(ctx, disabledConfig)
	assert.NoError(t, err)
	assert.IsType(t, &NoOpMemory{}, memory)
}

// TestGetInputOutputKeys tests the GetInputOutputKeys function.
func TestGetInputOutputKeys(t *testing.T) {
	// Test with empty maps
	_, _, err := GetInputOutputKeys(map[string]any{}, map[string]any{})
	assert.Error(t, err)

	// Test with valid inputs
	inputs := map[string]any{"input": "test input"}
	outputs := map[string]any{"output": "test output"}

	inputKey, outputKey, err := GetInputOutputKeys(inputs, outputs)
	assert.NoError(t, err)
	assert.Equal(t, "input", inputKey)
	assert.Equal(t, "output", outputKey)

	// Test with custom keys
	inputs = map[string]any{"query": "test query"}
	outputs = map[string]any{"answer": "test answer"}

	inputKey, outputKey, err = GetInputOutputKeys(inputs, outputs)
	assert.NoError(t, err)
	assert.Equal(t, "query", inputKey)
	assert.Equal(t, "answer", outputKey)
}
