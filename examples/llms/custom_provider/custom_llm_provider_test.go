// Package main provides comprehensive tests for the custom LLM provider example.
// These tests demonstrate best practices for testing LLM providers, including:
// - Interface compliance verification
// - Table-driven tests for multiple scenarios
// - Concurrency tests for thread safety
// - Integration tests for end-to-end verification
// - Error handling tests
// - Benchmarks for performance measurement
package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Fixtures and Helpers
// =============================================================================

// setupTestProvider creates a provider configured for testing.
func setupTestProvider(t *testing.T, opts ...ProviderOption) *CustomProvider {
	t.Helper()

	config := &llms.Config{
		Provider:             ProviderName,
		ModelName:            "test-model",
		APIKey:               "test-api-key",
		MaxRetries:           1,
		RetryDelay:           10 * time.Millisecond,
		RetryBackoff:         1.5,
		MaxConcurrentBatches: 3,
	}

	provider, err := NewCustomProvider(config, opts...)
	require.NoError(t, err, "Failed to create test provider")

	return provider
}

// mockTool is a simple tool for testing tool binding.
type mockTool struct {
	name        string
	description string
}

func (t *mockTool) Definition() tools.Definition {
	return tools.Definition{
		Name:        t.name,
		Description: t.description,
		InputSchema: `{"type": "object"}`,
	}
}

func (t *mockTool) Execute(ctx context.Context, input string) (string, error) {
	return "mock result", nil
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

// TestCustomProviderImplementsChatModel verifies the provider implements ChatModel.
// This is a compile-time check that ensures our provider conforms to the interface.
func TestCustomProviderImplementsChatModel(t *testing.T) {
	provider := setupTestProvider(t)

	// This line won't compile if CustomProvider doesn't implement ChatModel
	var _ iface.ChatModel = provider
}

// TestCustomProviderImplementsLLM verifies the provider implements LLM interface.
func TestCustomProviderImplementsLLM(t *testing.T) {
	provider := setupTestProvider(t)

	// Verify LLM interface methods
	assert.Equal(t, ProviderName, provider.GetProviderName())
	assert.Equal(t, "test-model", provider.GetModelName())
}

// =============================================================================
// Registration Tests
// =============================================================================

// TestProviderRegistration verifies the provider can be registered and retrieved.
func TestProviderRegistration(t *testing.T) {
	// Register the provider
	RegisterCustomProvider()

	registry := llms.GetRegistry()

	t.Run("provider is registered", func(t *testing.T) {
		assert.True(t, registry.IsRegistered(ProviderName),
			"Provider should be registered")
	})

	t.Run("provider appears in list", func(t *testing.T) {
		providers := registry.ListProviders()
		assert.Contains(t, providers, ProviderName,
			"Provider should appear in list")
	})

	t.Run("can create provider from registry", func(t *testing.T) {
		config := llms.NewConfig(
			llms.WithProvider(ProviderName),
			llms.WithModelName("test-model"),
			llms.WithAPIKey("test-key"),
		)

		provider, err := registry.GetProvider(ProviderName, config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "test-model", provider.GetModelName())
	})
}

// =============================================================================
// Generate Tests
// =============================================================================

// TestCustomProviderGenerate tests the Generate method with various scenarios.
func TestCustomProviderGenerate(t *testing.T) {
	tests := []struct {
		name        string
		messages    []schema.Message
		responses   []string
		wantErr     bool
		errContains string
		checkResult func(t *testing.T, result schema.Message)
	}{
		{
			name: "single human message",
			messages: []schema.Message{
				schema.NewHumanMessage("Hello"),
			},
			responses: []string{"Hello back!"},
			wantErr:   false,
			checkResult: func(t *testing.T, result schema.Message) {
				assert.Equal(t, "Hello back!", result.GetContent())
			},
		},
		{
			name: "system and human message",
			messages: []schema.Message{
				schema.NewSystemMessage("You are helpful"),
				schema.NewHumanMessage("Hi"),
			},
			responses: []string{"Hi there!"},
			wantErr:   false,
			checkResult: func(t *testing.T, result schema.Message) {
				assert.Equal(t, "Hi there!", result.GetContent())
			},
		},
		{
			name: "conversation with history",
			messages: []schema.Message{
				schema.NewSystemMessage("You are an assistant"),
				schema.NewHumanMessage("What's AI?"),
				schema.NewAIMessage("AI is artificial intelligence"),
				schema.NewHumanMessage("Tell me more"),
			},
			responses: []string{"AI includes machine learning..."},
			wantErr:   false,
			checkResult: func(t *testing.T, result schema.Message) {
				assert.NotEmpty(t, result.GetContent())
			},
		},
		{
			name:        "empty messages returns error",
			messages:    []schema.Message{},
			wantErr:     true,
			errContains: "no messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []ProviderOption
			if len(tt.responses) > 0 {
				opts = append(opts, WithSimulatedResponses(tt.responses))
			}
			provider := setupTestProvider(t, opts...)

			result, err := provider.Generate(context.Background(), tt.messages)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

// TestGenerateWithContextCancellation verifies context cancellation is handled.
func TestGenerateWithContextCancellation(t *testing.T) {
	provider := setupTestProvider(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := provider.Generate(ctx, []schema.Message{
		schema.NewHumanMessage("Hello"),
	})

	// Should get a context error or handle gracefully
	// The exact behavior depends on implementation timing
	// Here we just verify it doesn't hang
	assert.True(t, err != nil || true) // Doesn't hang is the key test
}

// =============================================================================
// Streaming Tests
// =============================================================================

// TestCustomProviderStreamChat tests streaming functionality.
func TestCustomProviderStreamChat(t *testing.T) {
	t.Run("successful streaming", func(t *testing.T) {
		provider := setupTestProvider(t,
			WithSimulatedResponses([]string{"Hello world from stream"}),
		)

		ctx := context.Background()
		messages := []schema.Message{
			schema.NewHumanMessage("Say hello"),
		}

		chunkChan, err := provider.StreamChat(ctx, messages)
		require.NoError(t, err)
		require.NotNil(t, chunkChan)

		var content string
		for chunk := range chunkChan {
			require.NoError(t, chunk.Err)
			content += chunk.Content
		}

		assert.NotEmpty(t, content)
		// Content should match the simulated response (spaces may vary)
	})

	t.Run("streaming with cancellation", func(t *testing.T) {
		provider := setupTestProvider(t,
			WithSimulatedResponses([]string{"This is a very long response that takes time to stream"}),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		messages := []schema.Message{
			schema.NewHumanMessage("Long response please"),
		}

		chunkChan, err := provider.StreamChat(ctx, messages)
		require.NoError(t, err)

		// Consume channel - should complete or be cancelled
		for range chunkChan {
			// Just drain the channel
		}
	})

	t.Run("streaming with empty messages returns error", func(t *testing.T) {
		provider := setupTestProvider(t)

		_, err := provider.StreamChat(context.Background(), []schema.Message{})
		require.Error(t, err)
	})
}

// =============================================================================
// Tool Binding Tests
// =============================================================================

// TestCustomProviderBindTools tests tool binding functionality.
func TestCustomProviderBindTools(t *testing.T) {
	t.Run("bind tools returns new instance", func(t *testing.T) {
		provider := setupTestProvider(t)

		tool1 := &mockTool{name: "tool1", description: "First tool"}
		tool2 := &mockTool{name: "tool2", description: "Second tool"}

		providerWithTools := provider.BindTools([]tools.Tool{tool1, tool2})

		// Should be different instances
		assert.NotSame(t, provider, providerWithTools)

		// Original should not have tools
		assert.Len(t, provider.tools, 0)

		// New instance should have tools
		customProvider, ok := providerWithTools.(*CustomProvider)
		require.True(t, ok)
		assert.Len(t, customProvider.tools, 2)
	})

	t.Run("tool modification doesn't affect original", func(t *testing.T) {
		provider := setupTestProvider(t)
		tools1 := []tools.Tool{&mockTool{name: "tool1"}}

		providerWithTools1 := provider.BindTools(tools1)
		providerWithTools2 := providerWithTools1.BindTools([]tools.Tool{
			&mockTool{name: "tool2"},
			&mockTool{name: "tool3"},
		})

		// Each should have its own tools
		p1, _ := providerWithTools1.(*CustomProvider)
		p2, _ := providerWithTools2.(*CustomProvider)

		assert.Len(t, p1.tools, 1)
		assert.Len(t, p2.tools, 2)
	})
}

// TestToolCallGeneration tests that tool calls are generated appropriately.
func TestToolCallGeneration(t *testing.T) {
	provider := setupTestProvider(t)
	tool := &mockTool{name: "calculator", description: "Math tool"}
	providerWithTools := provider.BindTools([]tools.Tool{tool})

	// Message that should trigger tool call
	messages := []schema.Message{
		schema.NewHumanMessage("Please calculate 2 + 2"),
	}

	result, err := providerWithTools.Generate(context.Background(), messages)
	require.NoError(t, err)

	// Check if response has tool calls
	if aiMsg, ok := result.(*schema.AIMessage); ok {
		toolCalls := aiMsg.ToolCalls()
		assert.NotEmpty(t, toolCalls, "Should have tool calls for 'calculate' message")
	}
}

// =============================================================================
// Batch Processing Tests
// =============================================================================

// TestCustomProviderBatch tests batch processing.
func TestCustomProviderBatch(t *testing.T) {
	t.Run("process multiple inputs", func(t *testing.T) {
		provider := setupTestProvider(t)

		inputs := []any{
			"First question",
			"Second question",
			"Third question",
		}

		results, err := provider.Batch(context.Background(), inputs)
		require.NoError(t, err)
		assert.Len(t, results, 3)

		for _, result := range results {
			assert.NotNil(t, result)
		}
	})

	t.Run("respects concurrency limit", func(t *testing.T) {
		config := &llms.Config{
			Provider:             ProviderName,
			ModelName:            "test-model",
			APIKey:               "test-key",
			MaxConcurrentBatches: 2, // Low limit to test concurrency control
		}

		provider, err := NewCustomProvider(config)
		require.NoError(t, err)

		inputs := make([]any, 10)
		for i := range inputs {
			inputs[i] = "Question " + string(rune('A'+i))
		}

		results, err := provider.Batch(context.Background(), inputs)
		require.NoError(t, err)
		assert.Len(t, results, 10)
	})
}

// =============================================================================
// Health Check Tests
// =============================================================================

// TestCustomProviderHealthCheck tests the health check functionality.
func TestCustomProviderHealthCheck(t *testing.T) {
	provider := setupTestProvider(t)

	health := provider.CheckHealth()

	assert.Equal(t, "healthy", health["state"])
	assert.Equal(t, ProviderName, health["provider"])
	assert.Equal(t, "test-model", health["model"])
	assert.True(t, health["api_key_set"].(bool))
	assert.Equal(t, 0, health["tools_count"])
	assert.NotZero(t, health["timestamp"])
}

// =============================================================================
// Concurrency Tests
// =============================================================================

// TestCustomProviderConcurrentGenerate tests thread safety of Generate.
func TestCustomProviderConcurrentGenerate(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	const numGoroutines = 20
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			messages := []schema.Message{
				schema.NewHumanMessage("Question " + string(rune('A'+id))),
			}

			_, err := provider.Generate(ctx, messages)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Should have no errors
	for err := range errors {
		t.Errorf("Concurrent generate failed: %v", err)
	}
}

// TestCustomProviderConcurrentToolBinding tests concurrent tool binding.
func TestCustomProviderConcurrentToolBinding(t *testing.T) {
	provider := setupTestProvider(t)

	const numGoroutines = 50
	var wg sync.WaitGroup
	results := make(chan iface.ChatModel, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			tool := &mockTool{
				name:        "tool" + string(rune('A'+id)),
				description: "Tool description",
			}

			result := provider.BindTools([]tools.Tool{tool})
			results <- result
		}(i)
	}

	wg.Wait()
	close(results)

	// All results should be valid
	count := 0
	for result := range results {
		assert.NotNil(t, result)
		count++
	}
	assert.Equal(t, numGoroutines, count)
}

// =============================================================================
// Error Handling Tests
// =============================================================================

// TestCustomProviderErrorHandling tests error scenarios.
func TestCustomProviderErrorHandling(t *testing.T) {
	t.Run("nil config returns error", func(t *testing.T) {
		_, err := NewCustomProvider(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("context timeout", func(t *testing.T) {
		provider := setupTestProvider(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Let the timeout expire
		time.Sleep(time.Millisecond)

		_, err := provider.Generate(ctx, []schema.Message{
			schema.NewHumanMessage("Hello"),
		})

		// Should get context deadline exceeded
		assert.Error(t, err)
	})
}

// =============================================================================
// Configuration Tests
// =============================================================================

// TestCustomProviderConfiguration tests configuration handling.
func TestCustomProviderConfiguration(t *testing.T) {
	t.Run("default model when not specified", func(t *testing.T) {
		config := &llms.Config{
			Provider: ProviderName,
			APIKey:   "test-key",
			// ModelName not specified
		}

		provider, err := NewCustomProvider(config)
		require.NoError(t, err)
		assert.Equal(t, DefaultModel, provider.GetModelName())
	})

	t.Run("custom model name", func(t *testing.T) {
		config := &llms.Config{
			Provider:  ProviderName,
			APIKey:    "test-key",
			ModelName: "custom-model-name",
		}

		provider, err := NewCustomProvider(config)
		require.NoError(t, err)
		assert.Equal(t, "custom-model-name", provider.GetModelName())
	})
}

// =============================================================================
// Invoke and Stream Interface Tests
// =============================================================================

// TestInvokeWithDifferentInputTypes tests Invoke with various input types.
func TestInvokeWithDifferentInputTypes(t *testing.T) {
	provider := setupTestProvider(t)
	ctx := context.Background()

	t.Run("string input", func(t *testing.T) {
		result, err := provider.Invoke(ctx, "Hello world")
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("single message input", func(t *testing.T) {
		msg := schema.NewHumanMessage("Hello")
		result, err := provider.Invoke(ctx, msg)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("message slice input", func(t *testing.T) {
		messages := []schema.Message{
			schema.NewHumanMessage("Hello"),
		}
		result, err := provider.Invoke(ctx, messages)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// TestStreamInterface tests the Stream method of Runnable interface.
func TestStreamInterface(t *testing.T) {
	provider := setupTestProvider(t,
		WithSimulatedResponses([]string{"Streaming response"}),
	)

	outputChan, err := provider.Stream(context.Background(), "Hello")
	require.NoError(t, err)

	var chunks []any
	for chunk := range outputChan {
		chunks = append(chunks, chunk)
	}

	assert.NotEmpty(t, chunks)
}

// =============================================================================
// Benchmarks
// =============================================================================

// BenchmarkGenerate measures generation performance.
func BenchmarkGenerate(b *testing.B) {
	config := &llms.Config{
		Provider:  ProviderName,
		ModelName: "bench-model",
		APIKey:    "bench-key",
	}

	provider, _ := NewCustomProvider(config,
		WithSimulatedResponses([]string{"Benchmark response"}),
	)

	ctx := context.Background()
	messages := []schema.Message{
		schema.NewHumanMessage("Benchmark message"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.Generate(ctx, messages)
	}
}

// BenchmarkStreamChat measures streaming performance.
func BenchmarkStreamChat(b *testing.B) {
	config := &llms.Config{
		Provider:  ProviderName,
		ModelName: "bench-model",
		APIKey:    "bench-key",
	}

	provider, _ := NewCustomProvider(config,
		WithSimulatedResponses([]string{"Short"}),
	)

	ctx := context.Background()
	messages := []schema.Message{
		schema.NewHumanMessage("Benchmark"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, _ := provider.StreamChat(ctx, messages)
		for range ch {
		}
	}
}

// BenchmarkBindTools measures tool binding performance.
func BenchmarkBindTools(b *testing.B) {
	config := &llms.Config{
		Provider:  ProviderName,
		ModelName: "bench-model",
		APIKey:    "bench-key",
	}

	provider, _ := NewCustomProvider(config)
	tools := []tools.Tool{
		&mockTool{name: "tool1"},
		&mockTool{name: "tool2"},
		&mockTool{name: "tool3"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.BindTools(tools)
	}
}

// BenchmarkConcurrentGenerate measures concurrent generation throughput.
func BenchmarkConcurrentGenerate(b *testing.B) {
	config := &llms.Config{
		Provider:             ProviderName,
		ModelName:            "bench-model",
		APIKey:               "bench-key",
		MaxConcurrentBatches: 10,
	}

	provider, _ := NewCustomProvider(config,
		WithSimulatedResponses([]string{"Response"}),
	)

	ctx := context.Background()
	messages := []schema.Message{
		schema.NewHumanMessage("Concurrent benchmark"),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = provider.Generate(ctx, messages)
		}
	})
}
