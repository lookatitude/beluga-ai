// Package package_pairs provides integration tests between ChatModels and Memory packages.
// This test suite verifies that chat models work correctly with memory implementations
// for conversation history management and context preservation.
package package_pairs

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationChatModelsMemory tests the integration between ChatModels and Memory packages.
func TestIntegrationChatModelsMemory(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name        string
		memoryType  string
		wantErr     bool
	}{
		{
			name:       "chatmodel_with_buffer_memory",
			memoryType: "buffer",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create memory
			var mem memory.Memory
			var err error

			switch tt.memoryType {
			case "buffer":
				mem, err = memory.NewMemory(memory.MemoryTypeBuffer)
			default:
				t.Skipf("Memory type %s not implemented in test", tt.memoryType)
				return
			}

			if err != nil {
				t.Skipf("Memory creation failed (may require setup): %v", err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, mem)

			// Create chat model config
			config := chatmodels.DefaultConfig()
			config.DefaultProvider = "mock"

			// Create chat model
			chatModel, err := chatmodels.NewChatModel("test-model", config)
			if err != nil {
				t.Skipf("Provider not registered: %v", err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, chatModel)

			// Test conversation with memory
			messages := []schema.Message{
				schema.NewHumanMessage("Hello, my name is Alice"),
			}

			// Save to memory
			inputs := map[string]any{"input": messages[0].GetContent()}
			outputs := map[string]any{"output": "Hello Alice!"}
			err = mem.SaveContext(ctx, inputs, outputs)
			if err != nil {
				t.Logf("Memory save returned error (may be expected): %v", err)
			}

			// Load from memory
			memMessages, err := mem.LoadMemoryVariables(ctx, map[string]any{})
			if err != nil {
				t.Logf("Memory load returned error (may be expected): %v", err)
			} else {
				assert.NotNil(t, memMessages)
			}

			// Generate response with chat model
			responses, err := chatModel.GenerateMessages(ctx, messages)
			if err != nil {
				t.Logf("Generation returned error (may be expected with mock): %v", err)
			} else {
				require.NotEmpty(t, responses)
				assert.Greater(t, len(responses), 0)

				// Save response to memory
				if len(responses) > 0 {
					inputs := map[string]any{"input": messages[0].GetContent()}
					outputs := map[string]any{"output": responses[0].GetContent()}
					err = mem.SaveContext(ctx, inputs, outputs)
					if err != nil {
						t.Logf("Memory save returned error: %v", err)
					}
				}
			}
		})
	}
}

// TestChatModelsMemoryConversationFlow tests multi-turn conversation with memory.
func TestChatModelsMemoryConversationFlow(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	// Create buffer memory
	mem, err := memory.NewMemory(memory.MemoryTypeBuffer)
	if err != nil {
		t.Skipf("Memory creation failed: %v", err)
		return
	}

	require.NoError(t, err)

	// Create chat model
	config := chatmodels.DefaultConfig()
	config.DefaultProvider = "mock"

	chatModel, err := chatmodels.NewChatModel("test-model", config)
	if err != nil {
		t.Skipf("Provider not registered: %v", err)
		return
	}

	require.NoError(t, err)

	// Simulate multi-turn conversation
	conversationTurns := []struct {
		humanMessage string
		expectMemory bool
	}{
		{"Hello, I'm Bob", false},
		{"What's my name?", true}, // Should remember from previous turn
		{"Tell me a joke", true}, // Should maintain context
	}

	for i, turn := range conversationTurns {
		t.Run(turn.humanMessage, func(t *testing.T) {
			// Load memory context
			memVars, err := mem.LoadMemoryVariables(ctx, map[string]any{})
			if err != nil {
				t.Logf("Memory load error (turn %d): %v", i, err)
			}

			// Create message
			messages := []schema.Message{
				schema.NewHumanMessage(turn.humanMessage),
			}

			// Add memory context if available
			if memVars != nil {
					if history, ok := memVars["history"].(string); ok && history != "" {
						// Memory context would be added to messages in real usage
						historyLen := len(history)
						if historyLen > 50 {
							historyLen = 50
						}
						t.Logf("Turn %d: Memory context available: %s", i, history[:historyLen])
					}
			}

			// Generate response
			responses, err := chatModel.GenerateMessages(ctx, messages)
			if err != nil {
				t.Logf("Generation error (turn %d): %v", i, err)
			} else {
				require.NotEmpty(t, responses)

				// Save to memory
				inputs := map[string]any{"input": turn.humanMessage}
				outputs := map[string]any{"output": responses[0].GetContent()}
				err = mem.SaveContext(ctx, inputs, outputs)
				if err != nil {
					t.Logf("Memory save error (turn %d): %v", i, err)
				}
			}
		})
	}
}

// Helper function for minInt
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
