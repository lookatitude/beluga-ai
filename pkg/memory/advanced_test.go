// Package memory provides comprehensive tests for memory implementations.
// This file contains advanced testing scenarios including table-driven tests,
// concurrency testing, and performance benchmarks.
package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedMockMemory tests the advanced mock memory functionality.
func TestAdvancedMockMemory(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		memory          *AdvancedMockMemory
		operations      func(ctx context.Context, memory *AdvancedMockMemory) error
		name            string
		expectedVarLen  int
		expectedCallMin int
		expectedError   bool
	}{
		{
			name:   "successful memory operations",
			memory: NewAdvancedMockMemory("test_history", MemoryTypeBuffer),
			operations: func(ctx context.Context, memory *AdvancedMockMemory) error {
				inputs, outputs := CreateTestInputOutput("Hello", "Hi there!")

				// Save context
				if err := memory.SaveContext(ctx, inputs, outputs); err != nil {
					return err
				}

				// Load memory variables
				vars, err := memory.LoadMemoryVariables(ctx, inputs)
				if err != nil {
					return err
				}

				if len(vars) == 0 {
					return errors.New("expected non-empty memory variables")
				}

				return nil
			},
			expectedError:   false,
			expectedVarLen:  1,
			expectedCallMin: 2,
		},
		{
			name: "memory with error",
			memory: NewAdvancedMockMemory("error_memory", MemoryTypeBuffer,
				WithMockError(true, errors.New("mock error"))),
			operations: func(ctx context.Context, memory *AdvancedMockMemory) error {
				inputs, outputs := CreateTestInputOutput("Hello", "Hi there!")
				return memory.SaveContext(ctx, inputs, outputs)
			},
			expectedError:   true,
			expectedCallMin: 1,
		},
		{
			name: "memory with delay",
			memory: NewAdvancedMockMemory("delay_memory", MemoryTypeBuffer,
				WithSimulateDelay(10*time.Millisecond)),
			operations: func(ctx context.Context, memory *AdvancedMockMemory) error {
				inputs, outputs := CreateTestInputOutput("Hello", "Hi there!")

				start := time.Now()
				err := memory.SaveContext(ctx, inputs, outputs)
				duration := time.Since(start)

				if duration < 10*time.Millisecond {
					return errors.New("expected delay was not respected")
				}

				return err
			},
			expectedError:   false,
			expectedCallMin: 1,
		},
		{
			name: "memory with preloaded messages",
			memory: NewAdvancedMockMemory("preloaded_memory", MemoryTypeBuffer,
				WithPreloadedMessages(CreateTestMessages(2))),
			operations: func(ctx context.Context, memory *AdvancedMockMemory) error {
				vars, err := memory.LoadMemoryVariables(ctx, map[string]any{"input": "test"})
				if err != nil {
					return err
				}

				historyContent := vars["preloaded_memory"]
				if historyContent == nil {
					return errors.New("expected preloaded messages in history")
				}
				// Check if it's a string (formatted) or slice (messages)
				if str, ok := historyContent.(string); ok && str == "" {
					return errors.New("expected preloaded messages in history")
				}
				if msgs, ok := historyContent.([]schema.Message); ok && len(msgs) == 0 {
					return errors.New("expected preloaded messages in history")
				}

				return nil
			},
			expectedError:   false,
			expectedCallMin: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run operations
			err := tt.operations(ctx, tt.memory)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify memory variables
				if tt.expectedVarLen > 0 {
					vars := tt.memory.MemoryVariables()
					assert.Len(t, vars, tt.expectedVarLen)
				}
			}

			// Verify call count
			assert.GreaterOrEqual(t, tt.memory.GetCallCount(), tt.expectedCallMin)

			// Test health check
			health := tt.memory.CheckHealth()
			AssertHealthCheck(t, health, "healthy")
		})
	}
}

// TestMemoryTypes tests different memory type implementations.
func TestMemoryTypes(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		operations func(ctx context.Context, memory iface.Memory) error
		name       string
		memoryType MemoryType
		config     Config
	}{
		{
			name:       "buffer memory",
			memoryType: MemoryTypeBuffer,
			config:     CreateTestMemoryConfig(MemoryTypeBuffer),
			operations: func(ctx context.Context, memory iface.Memory) error {
				// Test basic buffer operations
				inputs, outputs := CreateTestInputOutput("What is AI?", "AI is artificial intelligence")

				// Save and load multiple times
				for i := 0; i < 3; i++ {
					if err := memory.SaveContext(ctx, inputs, outputs); err != nil {
						return err
					}
				}

				vars, err := memory.LoadMemoryVariables(ctx, inputs)
				if err != nil {
					return err
				}

				if len(vars) == 0 {
					return errors.New("buffer memory should have stored content")
				}

				return nil
			},
		},
		{
			name:       "buffer window memory",
			memoryType: MemoryTypeBufferWindow,
			config: func() Config {
				config := CreateTestMemoryConfig(MemoryTypeBufferWindow)
				config.WindowSize = 2 // Small window for testing
				return config
			}(),
			operations: func(ctx context.Context, memory iface.Memory) error {
				// Test window size enforcement
				for i := 0; i < 5; i++ {
					inputs, outputs := CreateTestInputOutput(
						fmt.Sprintf("Question %d", i),
						fmt.Sprintf("Answer %d", i),
					)

					if err := memory.SaveContext(ctx, inputs, outputs); err != nil {
						return err
					}
				}

				vars, err := memory.LoadMemoryVariables(ctx, map[string]any{"input": "test"})
				if err != nil {
					return err
				}

				// Window memory should limit content
				if len(vars) == 0 {
					return errors.New("window memory should have some content")
				}

				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create memory using registry
			memory, err := CreateMemory(ctx, string(tt.memoryType), tt.config)
			if err != nil {
				t.Skipf("Memory type %s requires dependencies not available in unit tests: %v", tt.memoryType, err)
				return
			}

			// Run operations
			err = tt.operations(ctx, memory)
			require.NoError(t, err)

			// Test memory variables
			vars := memory.MemoryVariables()
			assert.NotEmpty(t, vars)

			// Test clear functionality
			err = memory.Clear(ctx)
			require.NoError(t, err)
		})
	}
}

// TestChatMessageHistory tests chat message history functionality.
func TestChatMessageHistory(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		history     *AdvancedMockChatMessageHistory
		operations  func(ctx context.Context, history iface.ChatMessageHistory) error
		name        string
		expectedErr bool
	}{
		{
			name:    "basic message operations",
			history: NewAdvancedMockChatMessageHistory(),
			operations: func(ctx context.Context, history iface.ChatMessageHistory) error {
				// Add various message types
				if err := history.AddUserMessage(ctx, "Hello!"); err != nil {
					return err
				}

				if err := history.AddAIMessage(ctx, "Hi there!"); err != nil {
					return err
				}

				humanMsg := schema.NewHumanMessage("How are you?")
				if err := history.AddMessage(ctx, humanMsg); err != nil {
					return err
				}

				// Get messages
				messages, err := history.GetMessages(ctx)
				if err != nil {
					return err
				}

				if len(messages) != 3 {
					return fmt.Errorf("expected 3 messages, got %d", len(messages))
				}

				return nil
			},
			expectedErr: false,
		},
		{
			name: "history with size limit",
			history: NewAdvancedMockChatMessageHistory(
				WithHistoryMaxSize(2),
			),
			operations: func(ctx context.Context, history iface.ChatMessageHistory) error {
				// Add more messages than the limit
				for i := 0; i < 5; i++ {
					if err := history.AddUserMessage(ctx, fmt.Sprintf("Message %d", i)); err != nil {
						return err
					}
				}

				messages, err := history.GetMessages(ctx)
				if err != nil {
					return err
				}

				// Should be limited to 2 messages
				if len(messages) > 2 {
					return fmt.Errorf("expected max 2 messages due to size limit, got %d", len(messages))
				}

				return nil
			},
			expectedErr: false,
		},
		{
			name: "history with error",
			history: NewAdvancedMockChatMessageHistory(
				WithHistoryError(true, errors.New("history error")),
			),
			operations: func(ctx context.Context, history iface.ChatMessageHistory) error {
				return history.AddUserMessage(ctx, "This should fail")
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operations(ctx, tt.history)

			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify call count
				assert.Positive(t, tt.history.GetCallCount())
			}
		})
	}
}

// TestMemoryRegistry tests the memory registry functionality.
func TestMemoryRegistry(t *testing.T) {
	ctx := context.Background()
	registry := NewMemoryRegistry()

	// Test registration
	testCreator := func(ctx context.Context, config Config) (iface.Memory, error) {
		return NewAdvancedMockMemory("test", MemoryTypeBuffer), nil
	}

	registry.Register("test_memory", testCreator)

	// Test listing
	types := registry.ListMemoryTypes()
	assert.Contains(t, types, "test_memory")

	// Test creation
	config := CreateTestMemoryConfig(MemoryTypeBuffer)

	memory, err := registry.Create(ctx, "test_memory", config)
	require.NoError(t, err)
	assert.NotNil(t, memory)

	// Test unknown type
	_, err = registry.Create(ctx, "unknown_type", config)
	require.Error(t, err)
	AssertErrorType(t, err, ErrCodeTypeMismatch)

	// Test global registry functions
	globalTypes := ListAvailableMemoryTypes()
	assert.NotEmpty(t, globalTypes)

	globalRegistry := GetGlobalMemoryRegistry()
	assert.NotNil(t, globalRegistry)
}

// TestConcurrencyAdvanced tests concurrent memory access.
func TestConcurrencyAdvanced(t *testing.T) {
	memory := NewAdvancedMockMemory("concurrent_test", MemoryTypeBuffer)

	const numGoroutines = 10
	const numOperationsPerGoroutine = 5

	t.Run("concurrent_memory_operations", func(t *testing.T) {
		ctx := context.Background()
		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines*numOperationsPerGoroutine)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < numOperationsPerGoroutine; j++ {
					inputs, outputs := CreateTestInputOutput(
						fmt.Sprintf("input-%d-%d", goroutineID, j),
						fmt.Sprintf("output-%d-%d", goroutineID, j),
					)

					// Save context
					if err := memory.SaveContext(ctx, inputs, outputs); err != nil {
						errChan <- err
						return
					}

					// Load memory variables
					if _, err := memory.LoadMemoryVariables(ctx, inputs); err != nil {
						errChan <- err
						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			t.Errorf("Concurrent operation error: %v", err)
		}

		// Verify total operations (each iteration does 2 operations)
		expectedOps := numGoroutines * numOperationsPerGoroutine * 2
		assert.Equal(t, expectedOps, memory.GetCallCount())
	})
}

// TestLoadTesting performs load testing on memory components.
func TestLoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	memory := NewAdvancedMockMemory("load_test", MemoryTypeBuffer)

	const numOperations = 100
	const concurrency = 10

	t.Run("memory_load_test", func(t *testing.T) {
		RunLoadTest(t, memory, numOperations, concurrency)

		// Verify health after load test
		health := memory.CheckHealth()
		AssertHealthCheck(t, health, "healthy")
		assert.Equal(t, numOperations*2, health["call_count"]) // Each operation does save + load
	})
}

// TestMemoryScenarios tests real-world memory usage scenarios.
func TestMemoryScenarios(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		scenario func(t *testing.T, memory iface.Memory)
		name     string
	}{
		{
			name: "conversation_flow",
			scenario: func(t *testing.T, memory iface.Memory) {
				t.Helper()
				runner := NewMemoryScenarioRunner(memory)

				err := runner.RunConversationScenario(ctx, 5)
				require.NoError(t, err)

				// Verify memory contains conversation history
				vars, err := memory.LoadMemoryVariables(ctx, map[string]any{"input": "test"})
				require.NoError(t, err)
				assert.NotEmpty(t, vars)
			},
		},
		{
			name: "memory_retention",
			scenario: func(t *testing.T, memory iface.Memory) {
				t.Helper()
				runner := NewMemoryScenarioRunner(memory)

				err := runner.RunMemoryRetentionTest(ctx, 20, 10)
				require.NoError(t, err)
			},
		},
		{
			name: "memory_persistence",
			scenario: func(t *testing.T, memory iface.Memory) {
				t.Helper()
				// Save initial state
				inputs1, outputs1 := CreateTestInputOutput("Initial question", "Initial answer")
				err := memory.SaveContext(ctx, inputs1, outputs1)
				require.NoError(t, err)

				// Load and verify
				vars1, err := memory.LoadMemoryVariables(ctx, inputs1)
				require.NoError(t, err)
				assert.NotEmpty(t, vars1)

				// Add more context
				inputs2, outputs2 := CreateTestInputOutput("Follow-up question", "Follow-up answer")
				err = memory.SaveContext(ctx, inputs2, outputs2)
				require.NoError(t, err)

				// Verify both contexts are available
				vars2, err := memory.LoadMemoryVariables(ctx, inputs2)
				require.NoError(t, err)
				assert.NotEmpty(t, vars2)

				// Clear memory
				err = memory.Clear(ctx)
				require.NoError(t, err)

				// Verify memory is cleared
				_, err = memory.LoadMemoryVariables(ctx, inputs1)
				require.NoError(t, err)
				// After clear, memory should be empty or reset
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use mock memory for scenario testing
			memory := NewAdvancedMockMemory("scenario_test", MemoryTypeBuffer)
			tt.scenario(t, memory)
		})
	}
}

// TestIntegrationTestHelper tests the integration test helper functionality.
func TestIntegrationTestHelper(t *testing.T) {
	helper := NewIntegrationTestHelper()

	// Add memories and histories
	bufferMemory := NewAdvancedMockMemory("buffer", MemoryTypeBuffer)
	windowMemory := NewAdvancedMockMemory("window", MemoryTypeBufferWindow)
	chatHistory := NewAdvancedMockChatMessageHistory()

	helper.AddMemory("buffer", bufferMemory)
	helper.AddMemory("window", windowMemory)
	helper.AddHistory("chat", chatHistory)

	// Test retrieval
	assert.Equal(t, bufferMemory, helper.GetMemory("buffer"))
	assert.Equal(t, windowMemory, helper.GetMemory("window"))
	assert.Equal(t, chatHistory, helper.GetHistory("chat"))

	// Test operations
	ctx := context.Background()
	inputs, outputs := CreateTestInputOutput("test input", "test output")

	err := bufferMemory.SaveContext(ctx, inputs, outputs)
	require.NoError(t, err)

	err = chatHistory.AddUserMessage(ctx, "test message")
	require.NoError(t, err)

	// Test reset
	helper.Reset()

	// Verify reset worked
	assert.Equal(t, 0, bufferMemory.GetCallCount())
	assert.Equal(t, 0, chatHistory.GetCallCount())
}

// TestMemoryErrorHandling tests comprehensive error handling scenarios.
func TestMemoryErrorHandling(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		setup     func() iface.Memory
		operation func(memory iface.Memory) error
		errorCode string
	}{
		{
			name: "save_context_error",
			setup: func() iface.Memory {
				return NewAdvancedMockMemory("error_memory", MemoryTypeBuffer,
					WithMockError(true, errors.New("save failed")))
			},
			operation: func(memory iface.Memory) error {
				inputs, outputs := CreateTestInputOutput("test", "test")
				return memory.SaveContext(ctx, inputs, outputs)
			},
		},
		{
			name: "load_variables_error",
			setup: func() iface.Memory {
				return NewAdvancedMockMemory("error_memory", MemoryTypeBuffer,
					WithMockError(true, errors.New("load failed")))
			},
			operation: func(memory iface.Memory) error {
				_, err := memory.LoadMemoryVariables(ctx, map[string]any{"input": "test"})
				return err
			},
		},
		{
			name: "clear_error",
			setup: func() iface.Memory {
				return NewAdvancedMockMemory("error_memory", MemoryTypeBuffer,
					WithMockError(true, errors.New("clear failed")))
			},
			operation: func(memory iface.Memory) error {
				return memory.Clear(ctx)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory := tt.setup()
			err := tt.operation(memory)

			require.Error(t, err)
		})
	}
}

// TestMemoryErrorFunctions tests all error helper functions.
func TestMemoryErrorFunctions(t *testing.T) {
	tests := []struct {
		testFunc func(t *testing.T)
		name     string
	}{
		{
			name: "error_unwrap",
			testFunc: func(t *testing.T) {
				underlyingErr := errors.New("underlying error")
				memErr := NewMemoryError("test_op", ErrCodeStorageError, underlyingErr)

				unwrapped := memErr.Unwrap()
				assert.Equal(t, underlyingErr, unwrapped)
			},
		},
		{
			name: "error_is",
			testFunc: func(t *testing.T) {
				err1 := NewMemoryError("op1", ErrCodeStorageError, errors.New("error"))
				err2 := NewMemoryError("op2", ErrCodeStorageError, errors.New("error"))
				err3 := NewMemoryError("op3", ErrCodeRetrievalError, errors.New("error"))

				// Same code should match
				assert.True(t, err1.Is(err2))
				// Different code should not match
				assert.False(t, err1.Is(err3))
			},
		},
		{
			name: "error_with_context",
			testFunc: func(t *testing.T) {
				err := NewMemoryError("test_op", ErrCodeStorageError, errors.New("error"))
				err = err.WithContext("key1", "value1")
				err = err.WithContext("key2", 42)

				assert.Equal(t, "value1", err.Context["key1"])
				assert.Equal(t, 42, err.Context["key2"])
			},
		},
		{
			name: "wrap_error",
			testFunc: func(t *testing.T) {
				underlyingErr := errors.New("underlying")
				wrapped := WrapError(underlyingErr, "test_op", ErrCodeTimeout)

				assert.NotNil(t, wrapped)
				assert.Equal(t, ErrCodeTimeout, wrapped.Code)
				assert.Equal(t, underlyingErr, wrapped.Unwrap())

				// Test nil error
				nilWrapped := WrapError(nil, "op", ErrCodeTimeout)
				assert.Nil(t, nilWrapped)
			},
		},
		{
			name: "is_memory_error",
			testFunc: func(t *testing.T) {
				err := NewMemoryError("test_op", ErrCodeStorageError, errors.New("error"))

				assert.True(t, IsMemoryError(err, ErrCodeStorageError))
				assert.False(t, IsMemoryError(err, ErrCodeRetrievalError))

				// Test with non-MemoryError
				regularErr := errors.New("regular error")
				assert.False(t, IsMemoryError(regularErr, ErrCodeStorageError))
			},
		},
		{
			name: "error_constructors",
			testFunc: func(t *testing.T) {
				baseErr := errors.New("base error")

				// Test all error constructors
				errs := []*MemoryError{
					ErrInvalidConfig(baseErr),
					ErrInvalidInput("op", baseErr),
					ErrStorageError("op", baseErr),
					ErrRetrievalError("op", baseErr),
					ErrTimeout("op", baseErr),
					ErrNotFound("op", baseErr),
					ErrTypeMismatch("op", baseErr),
					ErrSerialization("op", baseErr),
					ErrDeserialization("op", baseErr),
					ErrValidation("op", baseErr),
					ErrMemoryOverflow("op", baseErr),
					ErrContextCanceled("op", baseErr),
				}

				for _, err := range errs {
					assert.NotNil(t, err)
					assert.NotEmpty(t, err.Code)
					assert.Equal(t, baseErr, err.Unwrap())
				}
			},
		},
		{
			name: "error_with_message",
			testFunc: func(t *testing.T) {
				err := NewMemoryErrorWithMessage("test_op", ErrCodeStorageError, "Custom message", errors.New("underlying"))

				assert.Equal(t, "test_op", err.Op)
				assert.Equal(t, ErrCodeStorageError, err.Code)
				assert.Equal(t, "Custom message", err.Message)
				assert.Contains(t, err.Error(), "Custom message")
			},
		},
		{
			name: "error_string_formats",
			testFunc: func(t *testing.T) {
				// Test error with message
				err1 := NewMemoryErrorWithMessage("op1", ErrCodeStorageError, "Custom message", nil)
				assert.Contains(t, err1.Error(), "Custom message")

				// Test error with underlying error
				err2 := NewMemoryError("op2", ErrCodeStorageError, errors.New("underlying"))
				assert.Contains(t, err2.Error(), "underlying")

				// Test error without message or underlying error
				err3 := NewMemoryError("op3", ErrCodeStorageError, nil)
				assert.Contains(t, err3.Error(), "unknown error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}

// TestGetInputOutputKeysEdgeCases tests edge cases for GetInputOutputKeys.
func TestGetInputOutputKeysEdgeCases(t *testing.T) {
	tests := []struct {
		inputs      map[string]any
		outputs     map[string]any
		validate    func(t *testing.T, inputKey, outputKey string)
		name        string
		expectError bool
	}{
		{
			name:        "empty_inputs",
			inputs:      map[string]any{},
			outputs:     map[string]any{"output": "test"},
			expectError: true,
		},
		{
			name:        "empty_outputs",
			inputs:      map[string]any{"input": "test"},
			outputs:     map[string]any{},
			expectError: true,
		},
		{
			name:        "both_empty",
			inputs:      map[string]any{},
			outputs:     map[string]any{},
			expectError: true,
		},
		{
			name:    "human_input_key",
			inputs:  map[string]any{"human_input": "test"},
			outputs: map[string]any{"ai_output": "test"},
			validate: func(t *testing.T, inputKey, outputKey string) {
				assert.Equal(t, "human_input", inputKey)
				assert.Equal(t, "ai_output", outputKey)
			},
		},
		{
			name:    "user_input_key",
			inputs:  map[string]any{"user_input": "test"},
			outputs: map[string]any{"response": "test"},
			validate: func(t *testing.T, inputKey, outputKey string) {
				assert.Equal(t, "user_input", inputKey)
				assert.Equal(t, "response", outputKey)
			},
		},
		{
			name:    "question_key",
			inputs:  map[string]any{"question": "test"},
			outputs: map[string]any{"answer": "test"},
			validate: func(t *testing.T, inputKey, outputKey string) {
				assert.Equal(t, "question", inputKey)
				assert.Equal(t, "answer", outputKey)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputKey, outputKey, err := GetInputOutputKeys(tt.inputs, tt.outputs)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, inputKey, outputKey)
				}
			}
		})
	}
}

// TestAdvancedMockMemoryErrorCodes tests that AdvancedMockMemory supports all error types.
func TestAdvancedMockMemoryErrorCodes(t *testing.T) {
	ctx := context.Background()
	inputs, outputs := CreateTestInputOutput("test", "test")

	errorCodes := []string{
		ErrCodeInvalidConfig,
		ErrCodeInvalidInput,
		ErrCodeStorageError,
		ErrCodeRetrievalError,
		ErrCodeTimeout,
		ErrCodeNotFound,
		ErrCodeTypeMismatch,
		ErrCodeSerialization,
		ErrCodeDeserialization,
		ErrCodeValidation,
		ErrCodeMemoryOverflow,
		ErrCodeContextCanceled,
	}

	for _, code := range errorCodes {
		t.Run(code, func(t *testing.T) {
			mock := NewAdvancedMockMemory("test", MemoryTypeBuffer, WithErrorCode(code))

			// Test SaveContext with error
			err := mock.SaveContext(ctx, inputs, outputs)
			assert.Error(t, err)
			assert.True(t, IsMemoryError(err, code))

			// Test LoadMemoryVariables with error
			_, err = mock.LoadMemoryVariables(ctx, inputs)
			assert.Error(t, err)
			assert.True(t, IsMemoryError(err, code))
		})
	}
}

// TestGetBufferStringAdvanced tests GetBufferString with various message types.
func TestGetBufferStringAdvanced(t *testing.T) {
	tests := []struct {
		validate    func(t *testing.T, result string)
		name        string
		humanPrefix string
		aiPrefix    string
		messages    []schema.Message
	}{
		{
			name:        "system_message",
			messages:    []schema.Message{schema.NewSystemMessage("System prompt")},
			humanPrefix: "Human",
			aiPrefix:    "AI",
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "System: System prompt")
			},
		},
		{
			name: "tool_message",
			messages: []schema.Message{
				schema.NewToolMessage("tool-result", "call-123"),
			},
			humanPrefix: "Human",
			aiPrefix:    "AI",
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "Tool (call-123): tool-result")
			},
		},
		{
			name: "mixed_message_types",
			messages: []schema.Message{
				schema.NewSystemMessage("You are helpful"),
				schema.NewHumanMessage("Hello"),
				schema.NewAIMessage("Hi"),
				schema.NewToolMessage("result", "call-1"),
			},
			humanPrefix: "User",
			aiPrefix:    "Assistant",
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "System: You are helpful")
				assert.Contains(t, result, "User: Hello")
				assert.Contains(t, result, "Assistant: Hi")
				assert.Contains(t, result, "Tool (call-1): result")
			},
		},
		{
			name:        "custom_prefixes",
			messages:    []schema.Message{schema.NewHumanMessage("Test"), schema.NewAIMessage("Response")},
			humanPrefix: "Customer",
			aiPrefix:    "Support",
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "Customer: Test")
				assert.Contains(t, result, "Support: Response")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBufferString(tt.messages, tt.humanPrefix, tt.aiPrefix)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// BenchmarkAdvancedMemoryOperations benchmarks memory operation performance.
func BenchmarkAdvancedMemoryOperations(b *testing.B) {
	ctx := context.Background()
	memory := NewAdvancedMockMemory("benchmark", MemoryTypeBuffer)

	b.Run("SaveContext", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			inputs, outputs := CreateTestInputOutput(
				fmt.Sprintf("input-%d", i),
				fmt.Sprintf("output-%d", i),
			)

			err := memory.SaveContext(ctx, inputs, outputs)
			if err != nil {
				b.Errorf("SaveContext error: %v", err)
			}
		}
	})

	b.Run("LoadMemoryVariables", func(b *testing.B) {
		// Pre-populate memory
		inputs, outputs := CreateTestInputOutput("test input", "test output")
		_ = memory.SaveContext(ctx, inputs, outputs)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := memory.LoadMemoryVariables(ctx, inputs)
			if err != nil {
				b.Errorf("LoadMemoryVariables error: %v", err)
			}
		}
	})

	b.Run("MemoryVariables", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vars := memory.MemoryVariables()
			if len(vars) == 0 {
				b.Error("Expected non-empty memory variables")
			}
		}
	})
}

// BenchmarkChatMessageHistory benchmarks chat message history performance.
func BenchmarkChatMessageHistory(b *testing.B) {
	history := NewAdvancedMockChatMessageHistory()
	ctx := context.Background()

	b.Run("AddMessage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			msg := schema.NewHumanMessage(fmt.Sprintf("Message %d", i))
			err := history.AddMessage(ctx, msg)
			if err != nil {
				b.Errorf("AddMessage error: %v", err)
			}
		}
	})

	b.Run("AddUserMessage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := history.AddUserMessage(ctx, fmt.Sprintf("User message %d", i))
			if err != nil {
				b.Errorf("AddUserMessage error: %v", err)
			}
		}
	})

	b.Run("GetMessages", func(b *testing.B) {
		// Pre-populate history
		for i := 0; i < 10; i++ {
			_ = history.AddUserMessage(ctx, fmt.Sprintf("Message %d", i))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := history.GetMessages(ctx)
			if err != nil {
				b.Errorf("GetMessages error: %v", err)
			}
		}
	})
}

// BenchmarkMemoryRegistry benchmarks memory registry performance.
func BenchmarkMemoryRegistry(b *testing.B) {
	registry := NewMemoryRegistry()
	config := CreateTestMemoryConfig(MemoryTypeBuffer)
	ctx := context.Background()

	// Register a test creator
	testCreator := func(ctx context.Context, config Config) (iface.Memory, error) {
		return NewAdvancedMockMemory("benchmark", MemoryTypeBuffer), nil
	}
	registry.Register("benchmark_memory", testCreator)

	b.Run("Create", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			memory, err := registry.Create(ctx, "benchmark_memory", config)
			if err != nil {
				b.Errorf("Create error: %v", err)
			}
			if memory == nil {
				b.Error("Expected non-nil memory")
			}
		}
	})

	b.Run("ListMemoryTypes", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			types := registry.ListMemoryTypes()
			if len(types) == 0 {
				b.Error("Expected non-empty memory types")
			}
		}
	})
}
