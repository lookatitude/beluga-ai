package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockMemory is a mock implementation of the Memory interface for testing.
type MockMemory struct {
	MockGetMemoryKey        func() string
	MockLoadMemoryVariables func(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error)
	MockSaveContext         func(ctx context.Context, inputs map[string]interface{}, outputs map[string]string) error // MODIFIED HERE
	MockClear               func(ctx context.Context) error
	MockGetMemoryType       func() string
	MockGetMemoryVariables  func(ctx context.Context, inputs map[string]interface{}) ([]string, error)
}

func (m *MockMemory) GetMemoryKey() string {
	if m.MockGetMemoryKey != nil {
		return m.MockGetMemoryKey()
	}
	return "mock_memory_key"
}

func (m *MockMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
	if m.MockLoadMemoryVariables != nil {
		return m.MockLoadMemoryVariables(ctx, inputs)
	}
	return map[string]interface{}{"mock_loaded_var": "mock_value"}, nil
}

func (m *MockMemory) SaveContext(ctx context.Context, inputs map[string]interface{}, outputs map[string]string) error { // MODIFIED HERE
	if m.MockSaveContext != nil {
		return m.MockSaveContext(ctx, inputs, outputs) // MODIFIED HERE
	}
	return nil
}

func (m *MockMemory) Clear(ctx context.Context) error {
	if m.MockClear != nil {
		return m.MockClear(ctx)
	}
	return nil
}

// TestMemoryInterface ensures that MockMemory (and any future Memory implementations in this package)
// correctly implement the Memory interface.
func TestMemoryInterface(t *testing.T) {
	var _ Memory = (*MockMemory)(nil)

	// Example of how to use the mock for a basic test
	mockMem := &MockMemory{}
	ctx := context.Background() // ADDED HERE

	// Test GetMemoryKey
	assert.Equal(t, "mock_memory_key", mockMem.GetMemoryKey())

	// Test LoadMemoryVariables
	loadedVars, err := mockMem.LoadMemoryVariables(ctx, nil) // MODIFIED HERE
	assert.NoError(t, err)
	assert.NotNil(t, loadedVars)
	assert.Equal(t, "mock_value", loadedVars["mock_loaded_var"])

	// Test SaveContext
	err = mockMem.SaveContext(ctx, nil, nil) // MODIFIED HERE
	assert.NoError(t, err)

	// Test Clear
	err = mockMem.Clear(context.Background())
	assert.NoError(t, err)
}

// TestInputOutputKeys tests the InputOutputKeys struct.
// Assuming InputOutputKeys is defined in memory.go or a similar central file.
func TestInputOutputKeys(t *testing.T) {
	keys := InputOutputKeys{
		InputKey:  "input",
		OutputKey: "output",
	}
	assert.Equal(t, "input", keys.InputKey)
	assert.Equal(t, "output", keys.OutputKey)
}

// TestBaseMemory tests the BaseMemory struct and its methods.
// This assumes BaseMemory is a concrete type or an embeddable struct in memory.go
func TestBaseMemory(t *testing.T) {
	baseMem := NewBaseMemory("test_input_key", "test_output_key", "test_memory_key")
	assert.Equal(t, "test_input_key", baseMem.InputOutputKeys.InputKey)
	assert.Equal(t, "test_output_key", baseMem.InputOutputKeys.OutputKey)
	assert.Equal(t, "test_memory_key", baseMem.MemoryKey)

	// Test GetMemoryKey method of BaseMemory
	assert.Equal(t, "test_memory_key", baseMem.GetMemoryKey())
}

// TestGetInputOutputKeys tests the GetInputOutputKeys function.
func TestGetInputOutputKeys(t *testing.T) {
	// Case 1: No keys provided, use defaults
	defaultKeys := GetInputOutputKeys(nil, nil)
	assert.Equal(t, DefaultInputKey, defaultKeys.InputKey)
	assert.Equal(t, DefaultOutputKey, defaultKeys.OutputKey)

	// Case 2: InputKey provided, OutputKey default
	inputKey := "custom_input"
	customInputKeys := GetInputOutputKeys(&inputKey, nil)
	assert.Equal(t, "custom_input", customInputKeys.InputKey)
	assert.Equal(t, DefaultOutputKey, customInputKeys.OutputKey)

	// Case 3: OutputKey provided, InputKey default
	outputKey := "custom_output"
	customOutputKeys := GetInputOutputKeys(nil, &outputKey)
	assert.Equal(t, DefaultInputKey, customOutputKeys.InputKey)
	assert.Equal(t, "custom_output", customOutputKeys.OutputKey)

	// Case 4: Both keys provided
	bothKeys := GetInputOutputKeys(&inputKey, &outputKey)
	assert.Equal(t, "custom_input", bothKeys.InputKey)
	assert.Equal(t, "custom_output", bothKeys.OutputKey)
}

// TestGetPromptInputKey tests the GetPromptInputKey function.
func TestGetPromptInputKey(t *testing.T) {
	inputs := map[string]interface{}{
		"name": "Beluga",
		"age":  1,
	}
	memoryVariables := []string{"history"}

	// Case 1: InputKey exists and is not a memory variable
	inputKey := "name"
	promptInputKey, err := GetPromptInputKey(inputs, memoryVariables, &inputKey)
	assert.NoError(t, err)
	assert.Equal(t, "name", promptInputKey)

	// Case 2: InputKey exists and IS a memory variable (should error)
	inputKeyMem := "history"
	inputsWithHistory := map[string]interface{}{
		"history": "some chat history",
		"query":   "what is AI?",
	}
	_, err = GetPromptInputKey(inputsWithHistory, memoryVariables, &inputKeyMem)
	assert.Error(t, err) // Expecting an error

	// Case 3: InputKey is nil, multiple non-memory keys (should error)
	_, err = GetPromptInputKey(inputs, memoryVariables, nil)
	assert.Error(t, err)

	// Case 4: InputKey is nil, one non-memory key (should pick that one)
	singleInput := map[string]interface{}{"query": "what is AI?"}
	promptInputKey, err = GetPromptInputKey(singleInput, memoryVariables, nil)
	assert.NoError(t, err)
	assert.Equal(t, "query", promptInputKey)

	// Case 5: InputKey is nil, all keys are memory variables (should error)
	memoryOnlyInputs := map[string]interface{}{"history": "some chat history"}
	_, err = GetPromptInputKey(memoryOnlyInputs, memoryVariables, nil)
	assert.Error(t, err)

	// Case 6: InputKey provided but not in inputs (should error)
	missingKey := "non_existent_key"
	_, err = GetPromptInputKey(inputs, memoryVariables, &missingKey)
	assert.Error(t, err)
}




func (m *MockMemory) GetMemoryType() string {
	if m.MockGetMemoryType != nil {
		return m.MockGetMemoryType()
	}
	return "mock_memory_type"
}



func (m *MockMemory) GetMemoryVariables(ctx context.Context, inputs map[string]interface{}) ([]string, error) {
	if m.MockGetMemoryVariables != nil {
		return m.MockGetMemoryVariables(ctx, inputs)
	}
	return []string{"mock_memory_variable"}, nil
}

