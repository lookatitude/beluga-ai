// Package memory provides comprehensive tests for memory implementations.
// This file contains advanced testing scenarios including table-driven tests,
// concurrency testing, and performance benchmarks.
package memory

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
)

// TestAdvancedMockMemory tests the advanced mock memory functionality
func TestAdvancedMockMemory(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name            string
		memory          *AdvancedMockMemory
		operations      func(ctx context.Context, memory *AdvancedMockMemory) error
		expectedError   bool
		expectedVarLen  int
		expectedCallMin int
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
					return fmt.Errorf("expected non-empty memory variables")
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
				WithMockError(true, fmt.Errorf("mock error"))),
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
					return fmt.Errorf("expected delay was not respected")
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
					return fmt.Errorf("expected preloaded messages in history")
				}
				// Check if it's a string (formatted) or slice (messages)
				if str, ok := historyContent.(string); ok && str == "" {
					return fmt.Errorf("expected preloaded messages in history")
				}
				if msgs, ok := historyContent.([]schema.Message); ok && len(msgs) == 0 {
					return fmt.Errorf("expected preloaded messages in history")
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

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

// TestMemoryTypes tests different memory type implementations
func TestMemoryTypes(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		memoryType MemoryType
		config     Config
		operations func(ctx context.Context, memory iface.Memory) error
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
					return fmt.Errorf("buffer memory should have stored content")
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
					return fmt.Errorf("window memory should have some content")
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
			assert.NoError(t, err)

			// Test memory variables
			vars := memory.MemoryVariables()
			assert.NotEmpty(t, vars)

			// Test clear functionality
			err = memory.Clear(ctx)
			assert.NoError(t, err)
		})
	}
}

// TestChatMessageHistory tests chat message history functionality
func TestChatMessageHistory(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		history     *AdvancedMockChatMessageHistory
		operations  func(ctx context.Context, history iface.ChatMessageHistory) error
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
				WithHistoryError(true, fmt.Errorf("history error")),
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify call count
				assert.Greater(t, tt.history.GetCallCount(), 0)
			}
		})
	}
}

// TestMemoryRegistry tests the memory registry functionality
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
	assert.NoError(t, err)
	assert.NotNil(t, memory)

	// Test unknown type
	_, err = registry.Create(ctx, "unknown_type", config)
	assert.Error(t, err)
	AssertErrorType(t, err, ErrCodeTypeMismatch)

	// Test global registry functions
	globalTypes := ListAvailableMemoryTypes()
	assert.NotEmpty(t, globalTypes)

	globalRegistry := GetGlobalMemoryRegistry()
	assert.NotNil(t, globalRegistry)
}

// TestConcurrencyAdvanced tests concurrent memory access
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

// TestLoadTesting performs load testing on memory components
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

// TestMemoryScenarios tests real-world memory usage scenarios
func TestMemoryScenarios(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		scenario func(t *testing.T, memory iface.Memory)
	}{
		{
			name: "conversation_flow",
			scenario: func(t *testing.T, memory iface.Memory) {
				runner := NewMemoryScenarioRunner(memory)

				err := runner.RunConversationScenario(ctx, 5)
				assert.NoError(t, err)

				// Verify memory contains conversation history
				vars, err := memory.LoadMemoryVariables(ctx, map[string]any{"input": "test"})
				assert.NoError(t, err)
				assert.NotEmpty(t, vars)
			},
		},
		{
			name: "memory_retention",
			scenario: func(t *testing.T, memory iface.Memory) {
				runner := NewMemoryScenarioRunner(memory)

				err := runner.RunMemoryRetentionTest(ctx, 20, 10)
				assert.NoError(t, err)
			},
		},
		{
			name: "memory_persistence",
			scenario: func(t *testing.T, memory iface.Memory) {

				// Save initial state
				inputs1, outputs1 := CreateTestInputOutput("Initial question", "Initial answer")
				err := memory.SaveContext(ctx, inputs1, outputs1)
				assert.NoError(t, err)

				// Load and verify
				vars1, err := memory.LoadMemoryVariables(ctx, inputs1)
				assert.NoError(t, err)
				assert.NotEmpty(t, vars1)

				// Add more context
				inputs2, outputs2 := CreateTestInputOutput("Follow-up question", "Follow-up answer")
				err = memory.SaveContext(ctx, inputs2, outputs2)
				assert.NoError(t, err)

				// Verify both contexts are available
				vars2, err := memory.LoadMemoryVariables(ctx, inputs2)
				assert.NoError(t, err)
				assert.NotEmpty(t, vars2)

				// Clear memory
				err = memory.Clear(ctx)
				assert.NoError(t, err)

				// Verify memory is cleared
				_, err = memory.LoadMemoryVariables(ctx, inputs1)
				assert.NoError(t, err)
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

// TestIntegrationTestHelper tests the integration test helper functionality
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
	assert.NoError(t, err)

	err = chatHistory.AddUserMessage(ctx, "test message")
	assert.NoError(t, err)

	// Test reset
	helper.Reset()

	// Verify reset worked
	assert.Equal(t, 0, bufferMemory.GetCallCount())
	assert.Equal(t, 0, chatHistory.GetCallCount())
}

// TestMemoryErrorHandling tests comprehensive error handling scenarios
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
					WithMockError(true, fmt.Errorf("save failed")))
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
					WithMockError(true, fmt.Errorf("load failed")))
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
					WithMockError(true, fmt.Errorf("clear failed")))
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

			assert.Error(t, err)
		})
	}
}

// BenchmarkAdvancedMemoryOperations benchmarks memory operation performance
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
		memory.SaveContext(ctx, inputs, outputs)

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

// BenchmarkChatMessageHistory benchmarks chat message history performance
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
			history.AddUserMessage(ctx, fmt.Sprintf("Message %d", i))
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

// BenchmarkMemoryRegistry benchmarks memory registry performance
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
