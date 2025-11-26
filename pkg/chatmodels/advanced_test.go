// Package chatmodels provides comprehensive tests for chat model implementations.
// This file contains advanced testing scenarios including table-driven tests,
// concurrency testing, and performance benchmarks.
package chatmodels

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedMockChatModel tests the advanced mock chat model functionality.
func TestAdvancedMockChatModel(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		chatModel         *AdvancedMockChatModel
		name              string
		messages          []schema.Message
		expectedCallCount int
		expectedError     bool
		expectedResponse  bool
	}{
		{
			name:      "successful_generation",
			chatModel: NewAdvancedMockChatModel("test-model", "test-provider"),
			messages: []schema.Message{
				schema.NewHumanMessage("Hello, how are you?"),
			},
			expectedError:     false,
			expectedCallCount: 2, // Generate + StreamChat
			expectedResponse:  true,
		},
		{
			name: "chat_model_with_custom_responses",
			chatModel: NewAdvancedMockChatModel("custom-model", "custom-provider",
				WithMockResponses([]schema.Message{
					schema.NewAIMessage("Custom response 1"),
					schema.NewAIMessage("Custom response 2"),
				})),
			messages: []schema.Message{
				schema.NewHumanMessage("Test message 1"),
				schema.NewHumanMessage("Test message 2"),
			},
			expectedError:     false,
			expectedCallCount: 2,
			expectedResponse:  true,
		},
		{
			name: "chat_model_with_error",
			chatModel: NewAdvancedMockChatModel("error-model", "error-provider",
				WithMockError(true, errors.New("model service unavailable"))),
			messages: []schema.Message{
				schema.NewHumanMessage("This should fail"),
			},
			expectedError:     true,
			expectedCallCount: 1,
		},
		{
			name: "chat_model_with_delay",
			chatModel: NewAdvancedMockChatModel("delay-model", "delay-provider",
				WithStreamingDelay(20*time.Millisecond)),
			messages: []schema.Message{
				schema.NewHumanMessage("Test with delay"),
			},
			expectedError:     false,
			expectedCallCount: 2, // Generate + StreamChat
			expectedResponse:  true,
		},
		{
			name:              "multi_turn_conversation",
			chatModel:         NewAdvancedMockChatModel("conversation-model", "conversation-provider"),
			messages:          CreateTestMessages(3), // 3-turn conversation (6 messages)
			expectedError:     false,
			expectedCallCount: 2, // Generate + StreamChat
			expectedResponse:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Generate method
			var response schema.Message
			var err error

			if len(tt.messages) > 0 {
				start := time.Now()
				response, err = tt.chatModel.Generate(ctx, tt.messages)
				duration := time.Since(start)

				if tt.expectedError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					if tt.expectedResponse {
						AssertChatResponse(t, response, 5)
					}

					// Verify delay was respected if configured
					if tt.chatModel.streamingDelay > 0 {
						assert.GreaterOrEqual(t, duration, tt.chatModel.streamingDelay)
					}
				}
			}

			// Test streaming if no error expected
			if !tt.expectedError && len(tt.messages) > 0 {
				streamCh, err := tt.chatModel.StreamChat(ctx, tt.messages)
				require.NoError(t, err)

				chunks := make([]llmsiface.AIMessageChunk, 0)
				for chunk := range streamCh {
					chunks = append(chunks, chunk)
				}

				AssertStreamingResponse(t, chunks, 1)
			}

			// Verify call count
			assert.Equal(t, tt.expectedCallCount, tt.chatModel.GetCallCount())

			// Test health check
			health := tt.chatModel.CheckHealth()
			AssertChatModelHealth(t, health, "healthy")

			// Test conversation history
			history := tt.chatModel.GetConversationHistory()
			if !tt.expectedError && len(tt.messages) > 0 {
				assert.GreaterOrEqual(t, len(history), len(tt.messages),
					"Conversation history should include input messages")
			}

			// Test model metadata
			assert.Equal(t, tt.chatModel.modelName, tt.chatModel.GetModelName())
			assert.Equal(t, tt.chatModel.providerName, tt.chatModel.GetProviderName())
		})
	}
}

// TestChatModelStreaming tests streaming functionality specifically.
func TestChatModelStreaming(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name           string
		chatModel      *AdvancedMockChatModel
		messages       []schema.Message
		expectedChunks int
		streamingDelay time.Duration
		expectedError  bool
	}{
		{
			name:           "basic_streaming",
			chatModel:      NewAdvancedMockChatModel("stream-model", "stream-provider"),
			messages:       []schema.Message{schema.NewHumanMessage("Tell me a story")},
			expectedChunks: 3,
			streamingDelay: 0,
			expectedError:  false,
		},
		{
			name: "streaming_with_delay",
			chatModel: NewAdvancedMockChatModel("delay-stream-model", "delay-provider",
				WithStreamingDelay(10*time.Millisecond)),
			messages:       []schema.Message{schema.NewHumanMessage("Delayed stream test")},
			expectedChunks: 3,
			streamingDelay: 10 * time.Millisecond,
			expectedError:  false,
		},
		{
			name: "streaming_error",
			chatModel: NewAdvancedMockChatModel("error-stream-model", "error-provider",
				WithMockError(true, errors.New("streaming failed"))),
			messages:      []schema.Message{schema.NewHumanMessage("This should fail")},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			streamCh, err := tt.chatModel.StreamChat(ctx, tt.messages)

			if tt.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, streamCh)

			// Collect streaming chunks
			chunks := make([]llmsiface.AIMessageChunk, 0)
			chunkStart := time.Now()

			for chunk := range streamCh {
				chunks = append(chunks, chunk)

				// Verify no chunk errors
				require.NoError(t, chunk.Err, "Streaming chunk should not have error")
			}

			chunkDuration := time.Since(chunkStart)
			totalDuration := time.Since(start)

			// Verify streaming results
			assert.GreaterOrEqual(t, len(chunks), tt.expectedChunks,
				"Should receive at least %d chunks", tt.expectedChunks)

			// Verify streaming delay was respected
			if tt.streamingDelay > 0 {
				expectedMinDuration := tt.streamingDelay * time.Duration(len(chunks))
				assert.GreaterOrEqual(t, chunkDuration, expectedMinDuration)
			}

			// Verify chunks combine to form complete response
			fullContent := ""
			var fullContentSb225 strings.Builder
			for _, chunk := range chunks {
				_, _ = fullContentSb225.WriteString(chunk.Content)
			}
			fullContent += fullContentSb225.String()
			assert.NotEmpty(t, fullContent, "Combined streaming content should not be empty")

			t.Logf("Streaming test: %d chunks in %v (total: %v)",
				len(chunks), chunkDuration, totalDuration)
		})
	}
}

// TestChatModelConversationFlow tests multi-turn conversation scenarios.
func TestChatModelConversationFlow(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name                string
		conversationTurns   int
		contextPreservation bool
	}{
		{
			name:                "short_conversation",
			conversationTurns:   3,
			contextPreservation: true,
		},
		{
			name:                "long_conversation",
			conversationTurns:   10,
			contextPreservation: true,
		},
		{
			name:                "single_turn",
			conversationTurns:   1,
			contextPreservation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatModel := NewAdvancedMockChatModel("conversation-model", "conversation-provider")
			runner := NewChatModelScenarioRunner(chatModel)

			err := runner.RunConversationScenario(ctx, tt.conversationTurns)
			require.NoError(t, err)

			// Verify conversation history
			history := chatModel.GetConversationHistory()
			expectedMinMessages := tt.conversationTurns * 2 // Human + AI messages
			AssertConversationFlow(t, history, expectedMinMessages)

			// Verify call count
			assert.Equal(t, tt.conversationTurns, chatModel.GetCallCount())
		})
	}
}

// TestChatModelToolIntegration tests tool binding and usage.
func TestChatModelToolIntegration(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name           string
		toolCount      int
		toolsSupported bool
		expectedBound  bool
	}{
		{
			name:           "tools_supported",
			toolsSupported: true,
			toolCount:      3,
			expectedBound:  true,
		},
		{
			name:           "no_tools_support",
			toolsSupported: false,
			toolCount:      0,
			expectedBound:  true, // BindTools should still work but may ignore tools
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatModel := NewAdvancedMockChatModel("tool-model", "tool-provider",
				WithToolsSupport(tt.toolsSupported))

			// Create test tools
			testTools := make([]tools.Tool, tt.toolCount)
			for i := 0; i < tt.toolCount; i++ {
				tool := &tools.BaseTool{}
				tool.SetName(fmt.Sprintf("test_tool_%d", i+1))
				tool.SetDescription(fmt.Sprintf("Test tool %d for chat model testing", i+1))
				testTools[i] = tool
			}

			// Test tool binding
			if tt.toolCount > 0 {
				boundModel := chatModel.BindTools(testTools)
				assert.NotNil(t, boundModel, "BindTools should return a chat model")
				// Verify it's a different instance by checking pointer address
				// Use fmt.Sprintf to compare addresses since assert.NotEqual does deep comparison
				originalAddr := fmt.Sprintf("%p", chatModel)
				boundAddr := fmt.Sprintf("%p", boundModel)
				if originalAddr == boundAddr {
					t.Error("BindTools should return a new instance, but got the same pointer")
				}
			}

			// Test generation with potential tool calls
			messages := []schema.Message{
				schema.NewHumanMessage("Use the available tools to help me"),
			}

			response, err := chatModel.Generate(ctx, messages)
			require.NoError(t, err)
			assert.NotNil(t, response)
		})
	}
}

// TestConcurrencyAdvanced tests concurrent chat model operations.
func TestConcurrencyAdvanced(t *testing.T) {
	ctx := context.Background()
	chatModel := NewAdvancedMockChatModel("concurrent-test", "concurrent-provider")

	const numGoroutines = 8
	const operationsPerGoroutine = 4

	t.Run("concurrent_generation", func(t *testing.T) {
		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines*operationsPerGoroutine)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					messages := []schema.Message{
						schema.NewHumanMessage(fmt.Sprintf("Concurrent message %d-%d", goroutineID, j)),
					}

					_, err := chatModel.Generate(ctx, messages)
					if err != nil {
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
			t.Errorf("Concurrent generation error: %v", err)
		}

		// Verify total operations
		expectedOps := numGoroutines * operationsPerGoroutine
		assert.Equal(t, expectedOps, chatModel.GetCallCount())
	})

	t.Run("concurrent_streaming", func(t *testing.T) {
		streamModel := NewAdvancedMockChatModel("concurrent-stream", "stream-provider")

		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				messages := []schema.Message{
					schema.NewHumanMessage(fmt.Sprintf("Concurrent stream %d", goroutineID)),
				}

				streamCh, err := streamModel.StreamChat(ctx, messages)
				if err != nil {
					errChan <- err
					return
				}

				// Consume stream
				chunkCount := 0
				for chunk := range streamCh {
					if chunk.Err != nil {
						errChan <- chunk.Err
						return
					}
					chunkCount++
				}

				if chunkCount == 0 {
					errChan <- fmt.Errorf("no chunks received in stream %d", goroutineID)
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			t.Errorf("Concurrent streaming error: %v", err)
		}

		// Verify streaming calls
		assert.Equal(t, numGoroutines, streamModel.GetCallCount())
	})
}

// TestLoadTesting performs load testing on chat model components.
func TestLoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	chatModel := NewAdvancedMockChatModel("load-test", "load-provider")

	const numOperations = 50
	const concurrency = 8

	t.Run("chat_model_load_test", func(t *testing.T) {
		RunLoadTest(t, chatModel, numOperations, concurrency)

		// Verify health after load test
		health := chatModel.CheckHealth()
		AssertChatModelHealth(t, health, "healthy")
		assert.Equal(t, numOperations, health["call_count"])
	})
}

// TestChatModelScenarios tests real-world chat model usage scenarios.
func TestChatModelScenarios(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		scenario func(t *testing.T, chatModel llmsiface.ChatModel)
		name     string
	}{
		{
			name: "customer_service_bot",
			scenario: func(t *testing.T, chatModel llmsiface.ChatModel) {
				t.Helper()
				runner := NewChatModelScenarioRunner(chatModel)

				// Simulate customer service conversation
				err := runner.RunConversationScenario(ctx, 5)
				require.NoError(t, err)

				// Test specific customer service queries
				serviceQueries := []string{
					"I need help with my order",
					"Can you check my account status?",
					"I want to return a product",
				}

				err = runner.RunStreamingScenario(ctx, serviceQueries)
				require.NoError(t, err)
			},
		},
		{
			name: "educational_assistant",
			scenario: func(t *testing.T, chatModel llmsiface.ChatModel) {
				t.Helper()
				runner := NewChatModelScenarioRunner(chatModel)

				// Test educational conversation flow
				educationalQueries := []string{
					"Explain the concept of machine learning",
					"How do neural networks work?",
					"What are the applications of AI in healthcare?",
					"Can you give me examples of supervised learning?",
				}

				err := runner.RunStreamingScenario(ctx, educationalQueries)
				require.NoError(t, err)

				// Test follow-up conversation
				err = runner.RunConversationScenario(ctx, 3)
				require.NoError(t, err)
			},
		},
		{
			name: "code_assistant",
			scenario: func(t *testing.T, chatModel llmsiface.ChatModel) {
				t.Helper()
				// Test code-related conversations
				codeMessages := []schema.Message{
					schema.NewHumanMessage("Help me debug this Python function"),
					schema.NewHumanMessage("Explain this algorithm complexity"),
					schema.NewHumanMessage("Suggest improvements to this code structure"),
				}

				for i, msg := range codeMessages {
					response, err := chatModel.Generate(ctx, []schema.Message{msg})
					require.NoError(t, err, "Code query %d failed", i+1)
					AssertChatResponse(t, response, 10)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatModel := NewAdvancedMockChatModel("scenario-model", "scenario-provider")
			tt.scenario(t, chatModel)
		})
	}
}

// TestChatModelIntegrationTestHelper tests the integration test helper.
func TestChatModelIntegrationTestHelper(t *testing.T) {
	ctx := context.Background()
	helper := NewIntegrationTestHelper()

	// Add chat models
	gptModel := NewAdvancedMockChatModel("gpt-4", "openai")
	claudeModel := NewAdvancedMockChatModel("claude-3", "anthropic")

	helper.AddChatModel("gpt", gptModel)
	helper.AddChatModel("claude", claudeModel)

	// Test retrieval
	assert.Equal(t, gptModel, helper.GetChatModel("gpt"))
	assert.Equal(t, claudeModel, helper.GetChatModel("claude"))

	// Test operations
	messages := []schema.Message{schema.NewHumanMessage("Test message")}

	_, err := gptModel.Generate(ctx, messages)
	require.NoError(t, err)

	_, err = claudeModel.Generate(ctx, messages)
	require.NoError(t, err)

	// Test reset
	helper.Reset()

	// Verify reset worked
	assert.Equal(t, 0, gptModel.GetCallCount())
	assert.Equal(t, 0, claudeModel.GetCallCount())
	assert.Empty(t, gptModel.GetConversationHistory())
	assert.Empty(t, claudeModel.GetConversationHistory())
}

// TestChatModelErrorHandling tests comprehensive error handling scenarios.
func TestChatModelErrorHandling(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		setup     func() *AdvancedMockChatModel
		operation func(chatModel *AdvancedMockChatModel) error
		name      string
	}{
		{
			name: "generation_error",
			setup: func() *AdvancedMockChatModel {
				return NewAdvancedMockChatModel("error-model", "error-provider",
					WithMockError(true, errors.New("generation service down")))
			},
			operation: func(chatModel *AdvancedMockChatModel) error {
				messages := []schema.Message{schema.NewHumanMessage("Test")}
				_, err := chatModel.Generate(ctx, messages)
				return err
			},
		},
		{
			name: "streaming_error",
			setup: func() *AdvancedMockChatModel {
				return NewAdvancedMockChatModel("stream-error-model", "error-provider",
					WithMockError(true, errors.New("streaming service down")))
			},
			operation: func(chatModel *AdvancedMockChatModel) error {
				messages := []schema.Message{schema.NewHumanMessage("Test stream")}
				_, err := chatModel.StreamChat(ctx, messages)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatModel := tt.setup()
			err := tt.operation(chatModel)

			require.Error(t, err)
		})
	}
}

// BenchmarkChatModelOperations benchmarks chat model operation performance.
func BenchmarkChatModelOperations(b *testing.B) {
	ctx := context.Background()
	chatModel := NewAdvancedMockChatModel("benchmark-model", "benchmark-provider")
	messages := []schema.Message{
		schema.NewHumanMessage("Benchmark test message"),
	}

	b.Run("Generate", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := chatModel.Generate(ctx, messages)
			if err != nil {
				b.Errorf("Generate error: %v", err)
			}
		}
	})

	b.Run("StreamChat", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			streamCh, err := chatModel.StreamChat(ctx, messages)
			if err != nil {
				b.Errorf("StreamChat error: %v", err)
				continue
			}

			// Consume stream
			for chunk := range streamCh {
				if chunk.Err != nil {
					b.Errorf("Stream chunk error: %v", chunk.Err)
				}
			}
		}
	})

	b.Run("HealthCheck", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			health := chatModel.CheckHealth()
			if health == nil {
				b.Error("Health check should not return nil")
			}
		}
	})
}

// BenchmarkBenchmarkHelper tests the benchmark helper utility.
func BenchmarkBenchmarkHelper(b *testing.B) {
	chatModel := NewAdvancedMockChatModel("benchmark-helper", "helper-provider")
	helper := NewBenchmarkHelper(chatModel, 10)

	b.Run("BenchmarkGeneration", func(b *testing.B) {
		_, err := helper.BenchmarkGeneration(b.N)
		if err != nil {
			b.Errorf("BenchmarkGeneration error: %v", err)
		}
	})

	b.Run("BenchmarkStreaming", func(b *testing.B) {
		_, err := helper.BenchmarkStreaming(b.N)
		if err != nil {
			b.Errorf("BenchmarkStreaming error: %v", err)
		}
	})
}
