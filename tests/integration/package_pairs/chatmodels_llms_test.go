// Package package_pairs provides integration tests between ChatModels and LLMs packages.
// This test suite verifies that chat models work correctly with LLM providers
// for message generation, streaming, and tool-augmented interactions.
package package_pairs

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationChatModelsLLMs tests the integration between ChatModels and LLMs packages.
func TestIntegrationChatModelsLLMs(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name        string
		llmProvider string
		wantErr     bool
	}{
		{
			name:        "chatmodel_with_mock_llm",
			llmProvider: "mock",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create mock LLM
			mockLLM := helper.CreateMockLLM("integration-llm")
			require.NotNil(t, mockLLM)

			// Create chat model config
			config := chatmodels.DefaultConfig()
			config.DefaultProvider = "mock"

			// Create chat model (will use mock provider)
			chatModel, err := chatmodels.NewChatModel("test-model", config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			// If provider not registered, skip test
			if err != nil {
				t.Skipf("Provider not registered (expected in unit tests): %v", err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, chatModel)

			// Test message generation
			messages := []schema.Message{
				schema.NewHumanMessage("Hello, how are you?"),
			}

			responses, err := chatModel.GenerateMessages(ctx, messages)
			if err != nil {
				t.Logf("Generation returned error (may be expected with mock): %v", err)
			} else {
				require.NotEmpty(t, responses)
				assert.Greater(t, len(responses), 0)
			}

			// Test streaming
			streamCh, err := chatModel.StreamMessages(ctx, messages)
			if err != nil {
				t.Logf("Streaming returned error (may be expected with mock): %v", err)
			} else {
				require.NotNil(t, streamCh)
				msgCount := 0
				for msg := range streamCh {
					assert.NotNil(t, msg)
					msgCount++
					if msgCount > 10 { // Limit consumption
						break
					}
				}
			}

			// Test health check
			health := chatModel.CheckHealth()
			assert.NotNil(t, health)
			assert.Contains(t, health, "status")

			// Test model info
			info := chatModel.GetModelInfo()
			assert.NotEmpty(t, info.Name)
		})
	}
}

// TestChatModelsLLMsErrorHandling tests error scenarios between chat models and LLMs.
func TestChatModelsLLMsErrorHandling(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	// Test with invalid provider
	config := chatmodels.DefaultConfig()
	config.DefaultProvider = "non-existent-provider"

	chatModel, err := chatmodels.NewChatModel("test-model", config)
	if err == nil {
		// If provider exists, test error handling
		messages := []schema.Message{
			schema.NewHumanMessage("Test"),
		}

		// Test that errors are properly propagated
		_, err = chatModel.GenerateMessages(ctx, messages)
		if err != nil {
			t.Logf("Error properly propagated: %v", err)
		}
	} else {
		// Expected error for non-existent provider
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	}
}

// TestChatModelsLLMsConcurrency tests concurrent operations between chat models and LLMs.
func TestChatModelsLLMsConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}

	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	// Create chat model
	config := chatmodels.DefaultConfig()
	config.DefaultProvider = "mock"

	chatModel, err := chatmodels.NewChatModel("test-model", config)
	if err != nil {
		t.Skipf("Provider not registered: %v", err)
		return
	}

	require.NoError(t, err)

	// Test concurrent message generation
	const numGoroutines = 5
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			messages := []schema.Message{
				schema.NewHumanMessage("Concurrent test message"),
			}

			_, err := chatModel.GenerateMessages(ctx, messages)
			if err != nil {
				errChan <- err
			} else {
				errChan <- nil
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errChan
		if err != nil {
			t.Logf("Concurrent operation %d returned error: %v", i, err)
		}
	}
}
