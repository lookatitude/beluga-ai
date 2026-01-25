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
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
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

// TestNewDefaultConfig tests the NewDefaultConfig function.
func TestNewDefaultConfig(t *testing.T) {
	config := NewDefaultConfig()
	require.NotNil(t, config)
	assert.Equal(t, "gpt-3.5-turbo", config.DefaultModel)
	assert.Equal(t, "openai", config.DefaultProvider)
	assert.Equal(t, float32(0.7), config.DefaultTemperature)
	assert.Equal(t, 1000, config.DefaultMaxTokens)
	assert.Equal(t, 30*time.Second, config.DefaultTimeout)
}

// TestValidateConfig tests the ValidateConfig function.
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		config  *Config
		name    string
		wantErr bool
	}{
		{
			name:    "valid_config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid_temperature_too_high",
			config: func() *Config {
				c := DefaultConfig()
				c.DefaultTemperature = 3.0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid_temperature_negative",
			config: func() *Config {
				c := DefaultConfig()
				c.DefaultTemperature = -1.0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid_max_tokens",
			config: func() *Config {
				c := DefaultConfig()
				c.DefaultMaxTokens = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid_timeout",
			config: func() *Config {
				c := DefaultConfig()
				c.DefaultTimeout = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid_max_retries",
			config: func() *Config {
				c := DefaultConfig()
				c.DefaultMaxRetries = -1
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid_retry_delay",
			config: func() *Config {
				c := DefaultConfig()
				c.DefaultRetryDelay = -1 * time.Second
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid_max_concurrent_requests",
			config: func() *Config {
				c := DefaultConfig()
				c.MaxConcurrentRequests = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid_stream_buffer_size",
			config: func() *Config {
				c := DefaultConfig()
				c.StreamBufferSize = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid_stream_timeout",
			config: func() *Config {
				c := DefaultConfig()
				c.StreamTimeout = 0
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, IsValidationError(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGenerateMessagesConvenience tests the GenerateMessages convenience function.
func TestGenerateMessagesConvenience(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		chatModel   iface.ChatModel
		messages    []schema.Message
		expectedErr bool
	}{
		{
			name:      "successful_generation",
			chatModel: NewAdvancedMockChatModel("test-model", "test-provider"),
			messages: []schema.Message{
				schema.NewHumanMessage("Hello"),
			},
			expectedErr: false,
		},
		{
			name: "generation_error",
			chatModel: NewAdvancedMockChatModel("error-model", "error-provider",
				WithMockError(true, NewChatModelError("generate", "test-model", "test-provider", ErrCodeGeneration, errors.New("generation failed")))),
			messages: []schema.Message{
				schema.NewHumanMessage("Test"),
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateMessages(ctx, tt.chatModel, tt.messages)
			if tt.expectedErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

// TestStreamMessagesConvenience tests the StreamMessages convenience function.
func TestStreamMessagesConvenience(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		chatModel   iface.ChatModel
		messages    []schema.Message
		expectedErr bool
	}{
		{
			name:      "successful_streaming",
			chatModel: NewAdvancedMockChatModel("test-model", "test-provider"),
			messages: []schema.Message{
				schema.NewHumanMessage("Hello"),
			},
			expectedErr: false,
		},
		{
			name: "streaming_error",
			chatModel: NewAdvancedMockChatModel("error-model", "error-provider",
				WithMockError(true, NewChatModelError("stream", "test-model", "test-provider", ErrCodeStreaming, errors.New("streaming failed")))),
			messages: []schema.Message{
				schema.NewHumanMessage("Test"),
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streamCh, err := StreamMessages(ctx, tt.chatModel, tt.messages)
			if tt.expectedErr {
				require.Error(t, err)
				assert.Nil(t, streamCh)
			} else {
				require.NoError(t, err)
				require.NotNil(t, streamCh)
				// Consume stream
				msgCount := 0
				for msg := range streamCh {
					assert.NotNil(t, msg)
					msgCount++
				}
				assert.Positive(t, msgCount)
			}
		})
	}
}

// TestConfigFunctionalOptions tests all functional option functions.
func TestConfigFunctionalOptions(t *testing.T) {
	tests := []struct {
		option iface.Option
		verify func(t *testing.T, config map[string]any)
		name   string
	}{
		{
			name:   "WithTemperature",
			option: WithTemperature(0.8),
			verify: func(t *testing.T, config map[string]any) {
				assert.Equal(t, float32(0.8), config["temperature"])
			},
		},
		{
			name:   "WithMaxTokens",
			option: WithMaxTokens(2000),
			verify: func(t *testing.T, config map[string]any) {
				assert.Equal(t, 2000, config["max_tokens"])
			},
		},
		{
			name:   "WithTopP",
			option: WithTopP(0.9),
			verify: func(t *testing.T, config map[string]any) {
				assert.Equal(t, float32(0.9), config["top_p"])
			},
		},
		{
			name:   "WithStopSequences",
			option: WithStopSequences([]string{"stop1", "stop2"}),
			verify: func(t *testing.T, config map[string]any) {
				assert.Equal(t, []string{"stop1", "stop2"}, config["stop_sequences"])
			},
		},
		{
			name:   "WithSystemPrompt",
			option: WithSystemPrompt("You are a helpful assistant"),
			verify: func(t *testing.T, config map[string]any) {
				assert.Equal(t, "You are a helpful assistant", config["system_prompt"])
			},
		},
		{
			name:   "WithFunctionCalling",
			option: WithFunctionCalling(true),
			verify: func(t *testing.T, config map[string]any) {
				assert.Equal(t, true, config["function_calling"])
			},
		},
		{
			name:   "WithTimeout",
			option: WithTimeout(60 * time.Second),
			verify: func(t *testing.T, config map[string]any) {
				assert.Equal(t, 60*time.Second, config["timeout"])
			},
		},
		{
			name:   "WithMaxRetries",
			option: WithMaxRetries(5),
			verify: func(t *testing.T, config map[string]any) {
				assert.Equal(t, 5, config["max_retries"])
			},
		},
		{
			name:   "WithMetrics",
			option: WithMetrics(false),
			verify: func(t *testing.T, config map[string]any) {
				assert.Equal(t, false, config["enable_metrics"])
			},
		},
		{
			name:   "WithTracing",
			option: WithTracing(false),
			verify: func(t *testing.T, config map[string]any) {
				assert.Equal(t, false, config["enable_tracing"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config map[string]any
			tt.option.Apply(&config)
			require.NotNil(t, config)
			tt.verify(t, config)
		})
	}
}

// TestConfigGetProviderConfig tests GetProviderConfig method.
func TestConfigGetProviderConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		provider       string
		expectedConfig bool
		expectedErr    bool
	}{
		{
			name: "existing_provider_config",
			config: func() *Config {
				c := DefaultConfig()
				c.Providers["test-provider"] = &ProviderConfig{
					APIKey:     "test-key",
					Timeout:    60 * time.Second,
					MaxRetries: 5,
				}
				return c
			}(),
			provider:       "test-provider",
			expectedConfig: true,
			expectedErr:    false,
		},
		{
			name: "non_existent_provider_default",
			config: func() *Config {
				return DefaultConfig()
			}(),
			provider:       "non-existent",
			expectedConfig: true,
			expectedErr:    false,
		},
		{
			name: "invalid_provider_config_type",
			config: func() *Config {
				c := DefaultConfig()
				c.Providers["invalid"] = "not a ProviderConfig"
				return c
			}(),
			provider:       "invalid",
			expectedConfig: false,
			expectedErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := tt.config.GetProviderConfig(tt.provider)
			if tt.expectedErr {
				require.Error(t, err)
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
				if tt.provider == "test-provider" {
					assert.Equal(t, "test-key", config.APIKey)
					assert.Equal(t, 60*time.Second, config.Timeout)
					assert.Equal(t, 5, config.MaxRetries)
				}
			}
		})
	}
}

// TestChatModelErrorTypes tests all error types and helper functions.
func TestChatModelErrorTypes(t *testing.T) {
	t.Run("ChatModelError", func(t *testing.T) {
		err := NewChatModelError("test-op", "test-model", "test-provider", ErrCodeGeneration, errors.New("underlying error"))
		require.NotNil(t, err)
		assert.Equal(t, "test-op", err.Op)
		assert.Equal(t, "test-model", err.Fields["model"])
		assert.Equal(t, "test-provider", err.Fields["provider"])
		assert.Equal(t, ErrCodeGeneration, err.Code)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test-op")
		assert.Contains(t, err.Error(), ErrCodeGeneration)

		// Test WithField
		err = err.WithField("custom_field", "custom_value")
		assert.Equal(t, "custom_value", err.Fields["custom_field"])

		// Test Unwrap
		unwrapped := err.Unwrap()
		assert.Error(t, unwrapped)
	})

	t.Run("ValidationError", func(t *testing.T) {
		err := NewValidationError("test_field", "test message")
		require.NotNil(t, err)
		assert.Equal(t, "test_field", err.Field)
		assert.Equal(t, "test message", err.Message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test_field")
		assert.Contains(t, err.Error(), "test message")
	})

	t.Run("ProviderError", func(t *testing.T) {
		err := NewProviderError("test-provider", "test-operation", errors.New("provider error"))
		require.NotNil(t, err)
		assert.Equal(t, "test-provider", err.Provider)
		assert.Equal(t, "test-operation", err.Operation)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test-provider")
		assert.Contains(t, err.Error(), "test-operation")

		// Test Unwrap
		unwrapped := err.Unwrap()
		assert.Error(t, unwrapped)
	})

	t.Run("GenerationError", func(t *testing.T) {
		err := NewGenerationError("test-model", 5, errors.New("generation failed"))
		require.NotNil(t, err)
		assert.Equal(t, "test-model", err.Model)
		assert.Equal(t, 5, err.Messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test-model")
		assert.Contains(t, err.Error(), "5 messages")

		// Test WithTokenCount
		err = err.WithTokenCount(100)
		assert.Equal(t, 100, err.Tokens)
		assert.Contains(t, err.Error(), "100 tokens")

		// Test WithSuggestion
		err = err.WithSuggestion("Try reducing max_tokens")
		assert.Equal(t, "Try reducing max_tokens", err.Suggestion)
		assert.Contains(t, err.Error(), "Try reducing max_tokens")

		// Test Unwrap
		unwrapped := err.Unwrap()
		assert.Error(t, unwrapped)
	})

	t.Run("StreamingError", func(t *testing.T) {
		err := NewStreamingError("test-model", errors.New("streaming failed"))
		require.NotNil(t, err)
		assert.Equal(t, "test-model", err.Model)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test-model")

		// Test WithDuration
		err = err.WithDuration("5s")
		assert.Equal(t, "5s", err.Duration)
		assert.Contains(t, err.Error(), "5s")

		// Test Unwrap
		unwrapped := err.Unwrap()
		assert.Error(t, unwrapped)
	})
}

// TestErrorHelperFunctions tests error helper functions.
func TestErrorHelperFunctions(t *testing.T) {
	t.Run("IsRetryable", func(t *testing.T) {
		tests := []struct {
			err      error
			name     string
			expected bool
		}{
			{
				name:     "rate_limit_error",
				err:      NewChatModelError("op", "model", "provider", ErrCodeRateLimit, errors.New("rate limit")),
				expected: true,
			},
			{
				name:     "network_error",
				err:      NewChatModelError("op", "model", "provider", ErrCodeNetworkError, errors.New("network")),
				expected: true,
			},
			{
				name:     "timeout_error",
				err:      NewChatModelError("op", "model", "provider", ErrCodeTimeout, errors.New("timeout")),
				expected: true,
			},
			{
				name:     "resource_exhausted",
				err:      NewChatModelError("op", "model", "provider", ErrCodeResourceExhausted, errors.New("exhausted")),
				expected: true,
			},
			{
				name:     "generation_error",
				err:      NewChatModelError("op", "model", "provider", ErrCodeGeneration, errors.New("generation")),
				expected: false,
			},
			{
				name:     "common_timeout",
				err:      ErrTimeout,
				expected: true,
			},
			{
				name:     "common_rate_limit",
				err:      ErrRateLimitExceeded,
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := IsRetryable(tt.err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("IsValidationError", func(t *testing.T) {
		assert.True(t, IsValidationError(NewValidationError("field", "message")))
		assert.False(t, IsValidationError(errors.New("not validation")))
	})

	t.Run("IsProviderError", func(t *testing.T) {
		assert.True(t, IsProviderError(NewProviderError("provider", "op", errors.New("err"))))
		assert.False(t, IsProviderError(errors.New("not provider")))
	})

	t.Run("IsGenerationError", func(t *testing.T) {
		assert.True(t, IsGenerationError(NewGenerationError("model", 1, errors.New("err"))))
		assert.False(t, IsGenerationError(errors.New("not generation")))
	})

	t.Run("IsStreamingError", func(t *testing.T) {
		assert.True(t, IsStreamingError(NewStreamingError("model", errors.New("err"))))
		assert.False(t, IsStreamingError(errors.New("not streaming")))
	})

	t.Run("IsAuthenticationError", func(t *testing.T) {
		assert.True(t, IsAuthenticationError(NewChatModelError("op", "model", "provider", ErrCodeAuthentication, errors.New("auth"))))
		assert.True(t, IsAuthenticationError(ErrAuthenticationFailed))
		assert.False(t, IsAuthenticationError(errors.New("not auth")))
	})

	t.Run("IsQuotaError", func(t *testing.T) {
		assert.True(t, IsQuotaError(NewChatModelError("op", "model", "provider", ErrCodeQuotaExceeded, errors.New("quota"))))
		assert.True(t, IsQuotaError(ErrQuotaExceeded))
		assert.False(t, IsQuotaError(errors.New("not quota")))
	})
}

// TestChatModelErrorErrorMessages tests error message formatting.
func TestChatModelErrorErrorMessages(t *testing.T) {
	t.Run("error_with_model_and_provider", func(t *testing.T) {
		err := NewChatModelError("test-op", "test-model", "test-provider", ErrCodeGeneration, errors.New("underlying"))
		msg := err.Error()
		assert.Contains(t, msg, "test-model")
		assert.Contains(t, msg, "test-provider")
		assert.Contains(t, msg, "test-op")
		assert.Contains(t, msg, ErrCodeGeneration)
	})

	t.Run("error_with_model_only", func(t *testing.T) {
		err := NewChatModelError("test-op", "test-model", "", ErrCodeGeneration, errors.New("underlying"))
		msg := err.Error()
		assert.Contains(t, msg, "test-model")
		assert.Contains(t, msg, "test-op")
		assert.NotContains(t, msg, "provider:")
	})

	t.Run("error_without_model", func(t *testing.T) {
		err := NewChatModelError("test-op", "", "", ErrCodeGeneration, errors.New("underlying"))
		msg := err.Error()
		assert.Contains(t, msg, "test-op")
		assert.Contains(t, msg, ErrCodeGeneration)
	})
}
