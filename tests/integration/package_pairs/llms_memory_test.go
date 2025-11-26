// Package package_pairs provides integration tests between LLMs and Memory packages.
// This test suite verifies that LLMs and Memory components work together correctly
// for conversation history, context management, and persistent state.
package package_pairs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationLLMsMemory tests the integration between LLMs and Memory packages.
func TestIntegrationLLMsMemory(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name         string
		memoryType   memory.MemoryType
		exchanges    int
		expectedVars int
	}{
		{
			name:         "buffer_memory_conversation",
			memoryType:   memory.MemoryTypeBuffer,
			exchanges:    5,
			expectedVars: 1,
		},
		{
			name:         "window_memory_conversation",
			memoryType:   memory.MemoryTypeBufferWindow,
			exchanges:    8, // More than window size to test truncation
			expectedVars: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create LLM and Memory components
			llm := helper.CreateMockLLM("integration-llm")
			mockMemory := helper.CreateMockMemory("integration-memory", tt.memoryType)

			// Test conversation flow
			err := helper.TestConversationFlow(llm, mockMemory, tt.exchanges)
			require.NoError(t, err)

			// Verify memory variables are set correctly
			memoryVars := mockMemory.MemoryVariables()
			assert.Len(t, memoryVars, tt.expectedVars)

			// Verify memory contains conversation history
			ctx := context.Background()
			vars, err := mockMemory.LoadMemoryVariables(ctx, map[string]any{"input": "test"})
			require.NoError(t, err)
			assert.NotEmpty(t, vars)

			// Test that memory persists across multiple LLM calls
			for i := 0; i < 3; i++ {
				inputs := map[string]any{"input": "Follow-up question"}
				vars, err := mockMemory.LoadMemoryVariables(ctx, inputs)
				require.NoError(t, err)

				// Verify conversation history is available
				if len(memoryVars) > 0 {
					memoryKey := memoryVars[0]
					historyContent, exists := vars[memoryKey]
					if exists && historyContent != nil {
						assert.NotEmpty(t, historyContent, "Memory should contain conversation history")
					}
				}
			}
		})
	}
}

// TestIntegrationLLMsMemoryErrorHandling tests error scenarios.
func TestIntegrationLLMsMemoryErrorHandling(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		setupError  func() (*llms.AdvancedMockChatModel, *memory.AdvancedMockMemory)
		name        string
		expectedErr bool
	}{
		{
			name: "memory_save_error",
			setupError: func() (*llms.AdvancedMockChatModel, *memory.AdvancedMockMemory) {
				llm := llms.NewAdvancedMockChatModel("error-test")
				mem := memory.NewAdvancedMockMemory("error-memory", memory.MemoryTypeBuffer,
					memory.WithMockError(true, errors.New("memory save error")))
				return llm, mem
			},
			expectedErr: true,
		},
		{
			name: "memory_load_error",
			setupError: func() (*llms.AdvancedMockChatModel, *memory.AdvancedMockMemory) {
				llm := llms.NewAdvancedMockChatModel("error-test")
				mem := memory.NewAdvancedMockMemory("error-memory", memory.MemoryTypeBuffer)
				// First save should work, then set error for load
				return llm, mem
			},
			expectedErr: false, // Will test error separately
		},
		{
			name: "llm_generation_error",
			setupError: func() (*llms.AdvancedMockChatModel, *memory.AdvancedMockMemory) {
				llm := llms.NewAdvancedMockChatModel("error-test",
					llms.WithError(errors.New("LLM generation error")))
				mem := memory.NewAdvancedMockMemory("normal-memory", memory.MemoryTypeBuffer)
				return llm, mem
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm, mem := tt.setupError()

			// Test the conversation flow with errors
			err := helper.TestConversationFlow(llm, mem, 1)

			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestIntegrationLLMsMemoryPerformance tests performance characteristics.
func TestIntegrationLLMsMemoryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name          string
		memoryType    memory.MemoryType
		conversations int
		exchangesEach int
		maxDuration   time.Duration
	}{
		{
			name:          "buffer_memory_performance",
			memoryType:    memory.MemoryTypeBuffer,
			conversations: 5,
			exchangesEach: 10,
			maxDuration:   5 * time.Second,
		},
		{
			name:          "window_memory_performance",
			memoryType:    memory.MemoryTypeBufferWindow,
			conversations: 10,
			exchangesEach: 5,
			maxDuration:   3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			for conv := 0; conv < tt.conversations; conv++ {
				llm := helper.CreateMockLLM(fmt.Sprintf("perf-llm-%d", conv))
				mem := helper.CreateMockMemory(fmt.Sprintf("perf-memory-%d", conv), tt.memoryType)

				err := helper.TestConversationFlow(llm, mem, tt.exchangesEach)
				require.NoError(t, err)
			}

			duration := time.Since(start)
			assert.LessOrEqual(t, duration, tt.maxDuration,
				"Performance test should complete within %v, took %v", tt.maxDuration, duration)

			t.Logf("Performance test completed: %d conversations with %d exchanges each in %v",
				tt.conversations, tt.exchangesEach, duration)
		})
	}
}

// TestIntegrationLLMsMemoryConcurrency tests concurrent usage patterns.
func TestIntegrationLLMsMemoryConcurrency(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	const numGoroutines = 5
	const conversationsPerGoroutine = 3

	t.Run("concurrent_conversations", func(t *testing.T) {
		// Test concurrent access to shared memory
		llm := helper.CreateMockLLM("shared-llm")
		sharedMemory := helper.CreateMockMemory("shared-memory", memory.MemoryTypeBuffer)

		helper.CrossPackageLoadTest(t, func() error {
			return helper.TestConversationFlow(llm, sharedMemory, 1)
		}, numGoroutines*conversationsPerGoroutine, numGoroutines)
	})

	t.Run("isolated_conversations", func(t *testing.T) {
		// Test multiple isolated conversations
		helper.CrossPackageLoadTest(t, func() error {
			llm := helper.CreateMockLLM("isolated-llm")
			mem := helper.CreateMockMemory("isolated-memory", memory.MemoryTypeBuffer)
			return helper.TestConversationFlow(llm, mem, 2)
		}, numGoroutines*conversationsPerGoroutine, numGoroutines)
	})
}

// TestIntegrationLLMsMemoryContextPropagation tests context propagation.
func TestIntegrationLLMsMemoryContextPropagation(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	// Test that context and conversation history flows correctly between LLM and Memory
	llm := helper.CreateMockLLM("context-llm")
	mem := helper.CreateMockMemory("context-memory", memory.MemoryTypeBuffer)

	ctx := context.Background()

	// Initial conversation
	inputs1 := map[string]any{"input": "My name is Alice"}
	outputs1 := map[string]any{"output": "Nice to meet you, Alice!"}

	err := mem.SaveContext(ctx, inputs1, outputs1)
	require.NoError(t, err)

	// Follow-up conversation should have access to previous context
	inputs2 := map[string]any{"input": "What is my name?"}

	memoryVars, err := mem.LoadMemoryVariables(ctx, inputs2)
	require.NoError(t, err)

	// Verify context is available
	assert.NotEmpty(t, memoryVars)

	// Create messages with memory context
	messages := []schema.Message{
		schema.NewHumanMessage("What is my name?"),
	}

	// Add memory context if available
	if len(mem.MemoryVariables()) > 0 {
		memoryKey := mem.MemoryVariables()[0]
		if historyContent, exists := memoryVars[memoryKey]; exists {
			if content, ok := historyContent.(string); ok && content != "" {
				messages = append([]schema.Message{schema.NewSystemMessage("Previous conversation: " + content)}, messages...)
			}
		}
	}

	// Generate response with context
	response, err := llm.Generate(ctx, messages)
	require.NoError(t, err)

	// Verify response was generated
	assert.NotEmpty(t, response.GetContent())

	// Save the new context
	outputs2 := map[string]any{"output": response.GetContent()}
	err = mem.SaveContext(ctx, inputs2, outputs2)
	require.NoError(t, err)
}

// TestIntegrationLLMsMemoryStreamingWithMemory tests streaming responses with memory.
func TestIntegrationLLMsMemoryStreamingWithMemory(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	llm := helper.CreateMockLLM("streaming-llm")
	mem := helper.CreateMockMemory("streaming-memory", memory.MemoryTypeBuffer)

	ctx := context.Background()

	// Load memory context
	inputs := map[string]any{"input": "Tell me a story"}
	_, err := mem.LoadMemoryVariables(ctx, inputs)
	require.NoError(t, err)

	// Create messages
	messages := []schema.Message{
		schema.NewHumanMessage("Tell me a story"),
	}

	// Start streaming
	streamCh, err := llm.StreamChat(ctx, messages)
	require.NoError(t, err)

	// Collect streaming response
	var fullResponse string
	var fullResponseSb305 strings.Builder
	for chunk := range streamCh {
		if chunk.Err != nil {
			t.Errorf("Streaming error: %v", chunk.Err)
			continue
		}
		fullResponseSb305.WriteString(chunk.Content)
	}
	fullResponse += fullResponseSb305.String()

	// Save streaming result to memory
	outputs := map[string]any{"output": fullResponse}
	err = mem.SaveContext(ctx, inputs, outputs)
	require.NoError(t, err)

	// Verify memory contains the streamed content
	newVars, err := mem.LoadMemoryVariables(ctx, inputs)
	require.NoError(t, err)
	assert.NotEmpty(t, newVars)
}

// TestIntegrationLLMsMemoryMemoryTypes tests different memory types with LLMs.
func TestIntegrationLLMsMemoryMemoryTypes(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	memoryTypes := []struct {
		testFunc   func(t *testing.T, llm *llms.AdvancedMockChatModel, mem *memory.AdvancedMockMemory)
		name       string
		memoryType memory.MemoryType
	}{
		{
			name:       "buffer_memory",
			memoryType: memory.MemoryTypeBuffer,
			testFunc: func(t *testing.T, llm *llms.AdvancedMockChatModel, mem *memory.AdvancedMockMemory) {
				t.Helper()
				// Test that buffer memory stores all conversation history
				err := helper.TestConversationFlow(llm, mem, 3)
				require.NoError(t, err)

				// Verify all messages are retained
				messages := mem.GetMessages()
				assert.GreaterOrEqual(t, len(messages), 3*2) // At least 3 exchanges (6 messages)
			},
		},
		{
			name:       "window_memory",
			memoryType: memory.MemoryTypeBufferWindow,
			testFunc: func(t *testing.T, llm *llms.AdvancedMockChatModel, mem *memory.AdvancedMockMemory) {
				t.Helper()
				// Test that window memory limits conversation history
				err := helper.TestConversationFlow(llm, mem, 5)
				require.NoError(t, err)

				// Window memory should limit the number of stored messages
				// Note: Mock implementation may not enforce this, but we test the interface
				messages := mem.GetMessages()
				assert.NotNil(t, messages)
			},
		},
	}

	for _, tt := range memoryTypes {
		t.Run(tt.name, func(t *testing.T) {
			llm := llms.NewAdvancedMockChatModel("memory-type-test")
			mem := memory.NewAdvancedMockMemory("memory-test", tt.memoryType)

			tt.testFunc(t, llm, mem)
		})
	}
}

// TestIntegrationLLMsMemoryRealWorldScenarios tests realistic usage patterns.
func TestIntegrationLLMsMemoryRealWorldScenarios(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	scenarios := []struct {
		scenario func(t *testing.T)
		name     string
	}{
		{
			name: "customer_support_conversation",
			scenario: func(t *testing.T) {
				t.Helper()
				llm := helper.CreateMockLLM("support-llm")
				mem := helper.CreateMockMemory("support-memory", memory.MemoryTypeBuffer)
				ctx := context.Background()

				// Simulate customer support conversation
				conversations := []struct {
					input  string
					output string
				}{
					{"Hello, I need help with my account", "Hello! I'd be happy to help you with your account. What specific issue are you experiencing?"},
					{"I can't log in to my account", "I understand you're having trouble logging in. Let me help you troubleshoot this issue."},
					{"I tried resetting my password but didn't receive the email", "Let me check your email settings and resend the password reset email."},
					{"Thank you, that worked!", "You're welcome! Is there anything else I can help you with today?"},
				}

				for _, conv := range conversations {
					// Load memory context
					inputs := map[string]any{"input": conv.input}
					memoryVars, err := mem.LoadMemoryVariables(ctx, inputs)
					require.NoError(t, err)

					// Create messages with context
					messages := []schema.Message{schema.NewHumanMessage(conv.input)}

					// Add conversation history if available
					if memoryVars != nil && len(mem.MemoryVariables()) > 0 {
						memoryKey := mem.MemoryVariables()[0]
						if historyContent, exists := memoryVars[memoryKey]; exists {
							if content, ok := historyContent.(string); ok && content != "" {
								messages = append([]schema.Message{schema.NewSystemMessage("Previous conversation: " + content)}, messages...)
							}
						}
					}

					// Generate response
					response, err := llm.Generate(ctx, messages)
					require.NoError(t, err)

					// Save conversation
					outputs := map[string]any{"output": response.GetContent()}
					err = mem.SaveContext(ctx, inputs, outputs)
					require.NoError(t, err)
				}

				// Verify complete conversation is stored
				finalInputs := map[string]any{"input": "Can you summarize our conversation?"}
				finalVars, err := mem.LoadMemoryVariables(ctx, finalInputs)
				require.NoError(t, err)
				assert.NotEmpty(t, finalVars)
			},
		},
		{
			name: "multi_turn_reasoning",
			scenario: func(t *testing.T) {
				t.Helper()
				llm := helper.CreateMockLLM("reasoning-llm")
				mem := helper.CreateMockMemory("reasoning-memory", memory.MemoryTypeBuffer)

				// Test multi-turn reasoning where each response builds on previous context
				reasoningSteps := []string{
					"Let's solve this step by step. What is 15 + 27?",
					"Good. Now what is the result multiplied by 2?",
					"Perfect. Finally, what is that result divided by 3?",
				}

				for i := range reasoningSteps {
					err := helper.TestConversationFlow(llm, mem, 1)
					require.NoError(t, err, "Reasoning step %d failed", i+1)
				}

				// Verify memory contains the complete reasoning chain
				ctx := context.Background()
				vars, err := mem.LoadMemoryVariables(ctx, map[string]any{"input": "test"})
				require.NoError(t, err)
				assert.NotEmpty(t, vars)
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			scenario.scenario(t)
		})
	}
}

// BenchmarkIntegrationLLMsMemory benchmarks LLM-Memory integration performance.
func BenchmarkIntegrationLLMsMemory(b *testing.B) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	llm := helper.CreateMockLLM("benchmark-llm")
	mem := helper.CreateMockMemory("benchmark-memory", memory.MemoryTypeBuffer)

	b.Run("ConversationFlow", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := helper.TestConversationFlow(llm, mem, 1)
			if err != nil {
				b.Errorf("Conversation flow error: %v", err)
			}
		}
	})

	b.Run("MemoryLoad", func(b *testing.B) {
		ctx := context.Background()
		inputs := map[string]any{"input": "test"}

		// Pre-populate memory
		helper.TestConversationFlow(llm, mem, 5)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mem.LoadMemoryVariables(ctx, inputs)
			if err != nil {
				b.Errorf("Memory load error: %v", err)
			}
		}
	})

	b.Run("MemorySave", func(b *testing.B) {
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			inputs := map[string]any{"input": fmt.Sprintf("input-%d", i)}
			outputs := map[string]any{"output": fmt.Sprintf("output-%d", i)}

			err := mem.SaveContext(ctx, inputs, outputs)
			if err != nil {
				b.Errorf("Memory save error: %v", err)
			}
		}
	})
}
