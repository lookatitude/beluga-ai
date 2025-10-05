package memory

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/lookatitude/beluga-ai/pkg/schema"
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

// TestNoOpMemory tests the NoOpMemory implementation
func TestNoOpMemory(t *testing.T) {
	ctx := context.Background()
	noOpMem := &NoOpMemory{}

	// Test all methods
	variables := noOpMem.MemoryVariables()
	assert.Equal(t, []string{}, variables)

	vars, err := noOpMem.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{}, vars)

	err = noOpMem.SaveContext(ctx, map[string]any{}, map[string]any{})
	assert.NoError(t, err)

	err = noOpMem.Clear(ctx)
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

// TestFactory_AllMemoryTypes tests creating all supported memory types
func TestFactory_AllMemoryTypes(t *testing.T) {
	ctx := context.Background()
	factory := NewFactory()

	testCases := []struct {
		name        string
		memoryType  MemoryType
		expectError bool
	}{
		{"Buffer Memory", MemoryTypeBuffer, false},
		{"Buffer Window Memory", MemoryTypeBufferWindow, false},
		{"Summary Memory", MemoryTypeSummary, true},                             // Requires LLM dependency
		{"Summary Buffer Memory", MemoryTypeSummaryBuffer, true},                // Requires LLM dependency
		{"Vector Store Memory", MemoryTypeVectorStore, true},                    // Requires retriever dependency
		{"Vector Store Retriever Memory", MemoryTypeVectorStoreRetriever, true}, // Requires embedder/vectorstore dependencies
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Config{
				Type:    tc.memoryType,
				Enabled: true,
			}

			memory, err := factory.CreateMemory(ctx, config)
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, memory)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, memory)
			}
		})
	}
}

// TestFactory_ConfigurationValidation tests configuration validation
func TestFactory_ConfigurationValidation(t *testing.T) {
	ctx := context.Background()
	factory := NewFactory()

	testCases := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid configuration",
			config: Config{
				Type:    MemoryTypeBuffer,
				Enabled: true,
			},
			expectError: false,
		},
		{
			name: "Invalid memory type",
			config: Config{
				Type:    "invalid_type",
				Enabled: true,
			},
			expectError: true,
			errorMsg:    "invalid memory configuration",
		},
		{
			name: "Empty memory type",
			config: Config{
				Type:    "",
				Enabled: true,
			},
			expectError: true,
			errorMsg:    "invalid memory configuration",
		},
		{
			name: "Disabled memory",
			config: Config{
				Type:    MemoryTypeBuffer,
				Enabled: false,
			},
			expectError: false,
		},
		{
			name: "Valid with all fields",
			config: Config{
				Type:           MemoryTypeBuffer,
				MemoryKey:      "history",
				InputKey:       "input",
				OutputKey:      "output",
				ReturnMessages: true,
				WindowSize:     10,
				MaxTokenLimit:  2000,
				TopK:           5,
				HumanPrefix:    "Human",
				AIPrefix:       "AI",
				Enabled:        true,
				Timeout:        30 * time.Second,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			memory, err := factory.CreateMemory(ctx, tc.config)
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, memory)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				if tc.config.Enabled {
					assert.NotNil(t, memory)
				} else {
					assert.IsType(t, &NoOpMemory{}, memory)
				}
			}
		})
	}
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

// TestNewMemory tests the NewMemory convenience function
func TestNewMemory(t *testing.T) {
	testCases := []struct {
		name        string
		memoryType  MemoryType
		options     []Option
		expectError bool
	}{
		{
			name:        "Buffer memory with defaults",
			memoryType:  MemoryTypeBuffer,
			expectError: false,
		},
		{
			name:       "Buffer window memory with options",
			memoryType: MemoryTypeBufferWindow,
			options: []Option{
				WithMemoryKey("chat_history"),
				WithWindowSize(10),
				WithReturnMessages(true),
			},
			expectError: false,
		},
		{
			name:        "Unsupported memory type",
			memoryType:  "unsupported_type",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			memory, err := NewMemory(tc.memoryType, tc.options...)
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, memory)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, memory)
			}
		})
	}
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

	WithInputKey("query")(&config)
	assert.Equal(t, "query", config.InputKey)

	WithOutputKey("response")(&config)
	assert.Equal(t, "response", config.OutputKey)

	WithMaxTokenLimit(3000)(&config)
	assert.Equal(t, 3000, config.MaxTokenLimit)

	WithTopK(8)(&config)
	assert.Equal(t, 8, config.TopK)

	WithHumanPrefix("User")(&config)
	assert.Equal(t, "User", config.HumanPrefix)

	WithAIPrefix("Bot")(&config)
	assert.Equal(t, "Bot", config.AIPrefix)

	WithTimeout(60 * time.Second)(&config)
	assert.Equal(t, 60*time.Second, config.Timeout)
}

// TestConfiguration_Validation tests configuration validation
func TestConfiguration_Validation(t *testing.T) {
	validate := validator.New()

	testCases := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "Valid configuration",
			config: Config{
				Type:    MemoryTypeBuffer,
				Enabled: true,
			},
			expectError: false,
		},
		{
			name: "Invalid memory type",
			config: Config{
				Type:    "invalid",
				Enabled: true,
			},
			expectError: true,
		},
		{
			name: "Valid all types",
			config: Config{
				Type: MemoryTypeBuffer,
			},
			expectError: false,
		},
		{
			name: "Valid buffer window",
			config: Config{
				Type:       MemoryTypeBufferWindow,
				WindowSize: 10,
			},
			expectError: false,
		},
		{
			name: "Valid summary",
			config: Config{
				Type: MemoryTypeSummary,
			},
			expectError: false,
		},
		{
			name: "Valid vector store",
			config: Config{
				Type: MemoryTypeVectorStore,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validate.Struct(tc.config)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
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
	assert.Contains(t, err.Error(), "inputs map is empty")

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

	// Test with multiple possible keys
	inputs = map[string]any{"question": "test", "input": "also test"}
	outputs = map[string]any{"response": "test", "output": "also test"}

	inputKey, outputKey, err = GetInputOutputKeys(inputs, outputs)
	assert.NoError(t, err)
	assert.Equal(t, "input", inputKey)   // Should prefer "input" over "question"
	assert.Equal(t, "output", outputKey) // Should prefer "output" over "response"

	// Test fallback to first key
	inputs = map[string]any{"custom_input": "test"}
	outputs = map[string]any{"custom_output": "test"}

	inputKey, outputKey, err = GetInputOutputKeys(inputs, outputs)
	assert.NoError(t, err)
	assert.Equal(t, "custom_input", inputKey)
	assert.Equal(t, "custom_output", outputKey)
}

// TestGetBufferString tests the GetBufferString function
func TestGetBufferString(t *testing.T) {
	ctx := context.Background()

	// Create a history with some messages
	history := NewBaseChatMessageHistory()
	err := history.AddUserMessage(ctx, "Hello")
	require.NoError(t, err)
	err = history.AddAIMessage(ctx, "Hi there!")
	require.NoError(t, err)

	messages, err := history.GetMessages(ctx)
	require.NoError(t, err)

	result := GetBufferString(messages, "Human", "AI")

	assert.Contains(t, result, "Human: Hello")
	assert.Contains(t, result, "AI: Hi there!")

	// Test with custom prefixes
	result = GetBufferString(messages, "User", "Bot")
	assert.Contains(t, result, "User: Hello")
	assert.Contains(t, result, "Bot: Hi there!")

	// Test with empty messages
	result = GetBufferString([]schema.Message{}, "Human", "AI")
	assert.Equal(t, "", result)
}

// TestMemoryLifecycle tests the complete lifecycle of memory operations
func TestMemoryLifecycle(t *testing.T) {
	ctx := context.Background()

	// Create memory
	history := NewBaseChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	// Test initial state
	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)
	assert.Contains(t, vars, "history")

	// Add some context
	inputs1 := map[string]any{"input": "Hello"}
	outputs1 := map[string]any{"output": "Hi there!"}
	err = memory.SaveContext(ctx, inputs1, outputs1)
	assert.NoError(t, err)

	inputs2 := map[string]any{"input": "How are you?"}
	outputs2 := map[string]any{"output": "I'm doing well, thank you!"}
	err = memory.SaveContext(ctx, inputs2, outputs2)
	assert.NoError(t, err)

	// Load and verify
	vars, err = memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)

	historyStr, ok := vars["history"].(string)
	assert.True(t, ok)
	assert.Contains(t, historyStr, "Hello")
	assert.Contains(t, historyStr, "Hi there!")
	assert.Contains(t, historyStr, "How are you?")
	assert.Contains(t, historyStr, "I'm doing well")

	// Clear memory
	err = memory.Clear(ctx)
	assert.NoError(t, err)

	// Verify cleared
	vars, err = memory.LoadMemoryVariables(ctx, map[string]any{})
	assert.NoError(t, err)
	historyStr, ok = vars["history"].(string)
	assert.True(t, ok)
	assert.Equal(t, "", historyStr)
}

// TestMemoryErrorTypes tests custom error types
func TestMemoryErrorTypes(t *testing.T) {
	// Test creating different error types
	err1 := NewMemoryError("test_op", MemoryTypeBuffer, ErrCodeInvalidConfig, errors.New("config error"))
	assert.Equal(t, "memory test_op (buffer): config error", err1.Error())
	assert.Equal(t, "test_op", err1.Op)
	assert.Equal(t, MemoryTypeBuffer, err1.MemoryType)
	assert.Equal(t, ErrCodeInvalidConfig, err1.Code)

	// Test error wrapping
	wrappedErr := WrapError(err1, "save", MemoryTypeBufferWindow, ErrCodeStorageError)
	assert.NotNil(t, wrappedErr)
	assert.Equal(t, ErrCodeStorageError, wrappedErr.Code)

	// Test IsMemoryError
	assert.True(t, IsMemoryError(err1, ErrCodeInvalidConfig))
	assert.False(t, IsMemoryError(err1, ErrCodeStorageError))

	// Test error constructors
	err2 := ErrInvalidConfig(MemoryTypeSummary, errors.New("summary config error"))
	assert.Equal(t, ErrCodeInvalidConfig, err2.Code)
	assert.Equal(t, MemoryTypeSummary, err2.MemoryType)

	err3 := ErrStorageError("save_context", MemoryTypeVectorStore, errors.New("storage failed"))
	assert.Equal(t, ErrCodeStorageError, err3.Code)

	// Test WithContext
	err1.WithContext("key1", "value1")
	assert.Equal(t, "value1", err1.Context["key1"])
}

// TestConcurrentAccess tests concurrent access to memory
func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	history := NewBaseChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	// Test concurrent SaveContext operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			inputs := map[string]any{"input": "Message " + string(rune(id+'0'))}
			outputs := map[string]any{"output": "Response " + string(rune(id+'0'))}
			err := memory.SaveContext(ctx, inputs, outputs)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent LoadMemoryVariables operations
	for i := 0; i < 5; i++ {
		go func() {
			_, err := memory.LoadMemoryVariables(ctx, map[string]any{})
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}

// BenchmarkMemoryOperations benchmarks memory operations
func BenchmarkMemoryOperations(b *testing.B) {
	ctx := context.Background()
	history := NewBaseChatMessageHistory()
	memory := NewChatMessageBufferMemory(history)

	inputs := map[string]any{"input": "Hello world"}
	outputs := map[string]any{"output": "Hi there!"}

	b.Run("SaveContext", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			memory.SaveContext(ctx, inputs, outputs)
		}
	})

	b.Run("LoadMemoryVariables", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			memory.LoadMemoryVariables(ctx, map[string]any{})
		}
	})

	b.Run("SaveAndLoad", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			memory.SaveContext(ctx, inputs, outputs)
			memory.LoadMemoryVariables(ctx, map[string]any{})
		}
	})
}

// BenchmarkFactory benchmarks factory creation
func BenchmarkFactory(b *testing.B) {
	ctx := context.Background()
	factory := NewFactory()
	config := Config{
		Type:      MemoryTypeBuffer,
		Enabled:   true,
		MemoryKey: "history",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memory, err := factory.CreateMemory(ctx, config)
		if err != nil {
			b.Fatal(err)
		}
		_ = memory
	}
}

// TestMemoryIntegration tests integration scenarios
func TestMemoryIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("BufferMemoryWorkflow", func(t *testing.T) {
		history := NewBaseChatMessageHistory()
		memory := NewChatMessageBufferMemory(history)

		// Simulate a conversation
		conversation := []struct {
			input  string
			output string
		}{
			{"Hello", "Hi there!"},
			{"How are you?", "I'm doing well, thank you!"},
			{"What's the weather like?", "I don't have access to weather data."},
		}

		// Save conversation
		for _, msg := range conversation {
			inputs := map[string]any{"input": msg.input}
			outputs := map[string]any{"output": msg.output}
			err := memory.SaveContext(ctx, inputs, outputs)
			assert.NoError(t, err)
		}

		// Load and verify
		vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		historyStr := vars["history"].(string)
		for _, msg := range conversation {
			assert.Contains(t, historyStr, msg.input)
			assert.Contains(t, historyStr, msg.output)
		}

		// Verify message order
		lines := strings.Split(strings.TrimSpace(historyStr), "\n")
		assert.Len(t, lines, len(conversation)*2)
	})

	t.Run("WindowMemoryWorkflow", func(t *testing.T) {
		history := NewBaseChatMessageHistory()
		memory := NewConversationBufferWindowMemory(history, 3, "history", false)

		// Add more messages than window size
		for i := 0; i < 5; i++ {
			inputs := map[string]any{"input": "Message " + string(rune(i+'0'))}
			outputs := map[string]any{"output": "Response " + string(rune(i+'0'))}
			err := memory.SaveContext(ctx, inputs, outputs)
			assert.NoError(t, err)
		}

		// Load and verify only last 3 interactions are kept
		vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		historyStr := vars["history"].(string)
		assert.Contains(t, historyStr, "Message 2")
		assert.Contains(t, historyStr, "Message 3")
		assert.Contains(t, historyStr, "Message 4")
		assert.NotContains(t, historyStr, "Message 0")
		assert.NotContains(t, historyStr, "Message 1")
	})
}
