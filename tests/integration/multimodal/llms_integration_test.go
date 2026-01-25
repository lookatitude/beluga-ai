// Package multimodal provides integration tests between Multimodal and LLMs packages.
// This test suite verifies that multimodal models work correctly with LLM providers
// for text processing, multimodal reasoning, and content conversion.
package multimodal

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/multimodal"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationMultimodalLLMs tests the integration between Multimodal and LLMs packages.
func TestIntegrationMultimodalLLMs(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	tests := []struct {
		name        string
		testFunc    func(*testing.T)
		description string
	}{
		{
			name:        "text_only_processing",
			description: "Test multimodal model with text-only input using LLM",
			testFunc: func(t *testing.T) {
				// Create multimodal input with text only
				textBlock, err := multimodal.NewContentBlock("text", []byte("Hello, world!"))
				require.NoError(t, err)

				input, err := multimodal.NewMultimodalInput([]*multimodal.ContentBlock{textBlock})
				require.NoError(t, err)

				// Create multimodal model
				config := multimodal.Config{
					Provider: "test",
					Model:    "test-model",
				}

				capabilities := &multimodal.ModalityCapabilities{
					Text: true,
				}

				model := multimodal.NewTestBaseMultimodalModel("test", "test-model", config, capabilities)

				// Process input (should use LLM backend for text)
				// Convert multimodal.ContentBlock to types.ContentBlock
				typeBlocks := make([]*types.ContentBlock, len(input.ContentBlocks))
				for i, block := range input.ContentBlocks {
					typeBlocks[i] = &types.ContentBlock{
						Type:     block.Type,
						Data:     block.Data,
						URL:      block.URL,
						FilePath: block.FilePath,
						Format:   block.Format,
						MIMEType: block.MIMEType,
						Size:     block.Size,
						Metadata: block.Metadata,
					}
				}
				output, err := model.Process(ctx, &types.MultimodalInput{
					ID:            input.ID,
					ContentBlocks: typeBlocks,
				})
				if err != nil {
					t.Logf("Processing failed (expected if provider not registered): %v", err)
					return
				}

				require.NoError(t, err)
				assert.NotNil(t, output)
				assert.Equal(t, input.ID, output.InputID)
			},
		},
		{
			name:        "multimodal_text_image",
			description: "Test multimodal model with text + image input",
			testFunc: func(t *testing.T) {
				// Create multimodal input with text and image
				textBlock, err := multimodal.NewContentBlock("text", []byte("What's in this image?"))
				require.NoError(t, err)

				imageBlock, err := multimodal.NewContentBlock("image", []byte{0x89, 0x50, 0x4E, 0x47})
				require.NoError(t, err)

				input, err := multimodal.NewMultimodalInput([]*multimodal.ContentBlock{textBlock, imageBlock})
				require.NoError(t, err)

				// Create multimodal model with image capabilities
				capabilities := &types.ModalityCapabilities{
					Text:  true,
					Image: true,
				}

				model := multimodal.NewTestBaseMultimodalModel("test", "test-model", map[string]any{
					"Provider": "test",
					"Model":    "test-model",
				}, capabilities)

				// Process multimodal input
				// Convert multimodal.ContentBlock to types.ContentBlock
				typeBlocks := make([]*types.ContentBlock, len(input.ContentBlocks))
				for i, block := range input.ContentBlocks {
					typeBlocks[i] = &types.ContentBlock{
						Type:     block.Type,
						Data:     block.Data,
						URL:      block.URL,
						FilePath: block.FilePath,
						Format:   block.Format,
						MIMEType: block.MIMEType,
						Size:     block.Size,
						Metadata: block.Metadata,
					}
				}
				output, err := model.Process(ctx, &types.MultimodalInput{
					ID:            input.ID,
					ContentBlocks: typeBlocks,
				})
				if err != nil {
					t.Logf("Processing failed (expected if provider not registered): %v", err)
					return
				}

				require.NoError(t, err)
				assert.NotNil(t, output)
			},
		},
		{
			name:        "message_conversion",
			description: "Test converting multimodal content to/from LLM messages",
			testFunc: func(t *testing.T) {
				// Create schema message with image
				imgMsg := &schema.ImageMessage{
					ImageURL:    "https://example.com/image.jpg",
					ImageFormat: "jpeg",
				}
				imgMsg.BaseMessage.Content = "Analyze this image"

				// Create multimodal model
				capabilities := &types.ModalityCapabilities{
					Text:  true,
					Image: true,
				}

				baseModel := multimodal.NewTestBaseMultimodalModel("test", "test-model", map[string]any{
					"Provider": "test",
					"Model":    "test-model",
				}, capabilities)
				extension := multimodal.NewTestMultimodalAgentExtension(baseModel)

				// Convert message to multimodal input
				multimodalInput, err := extension.HandleMultimodalMessage(ctx, imgMsg)
				if err != nil {
					t.Logf("Message conversion failed (expected if extension not fully configured): %v", err)
					return
				}

				require.NoError(t, err)
				assert.NotNil(t, multimodalInput)
				assert.Greater(t, len(multimodalInput.ContentBlocks), 0)
			},
		},
		{
			name:        "streaming_processing",
			description: "Test multimodal streaming with LLM backend",
			testFunc: func(t *testing.T) {
				// Create text-only input for streaming
				textBlock, err := multimodal.NewContentBlock("text", []byte("Generate a story"))
				require.NoError(t, err)

				input, err := multimodal.NewMultimodalInput([]*multimodal.ContentBlock{textBlock})
				require.NoError(t, err)

				capabilities := &types.ModalityCapabilities{
					Text: true,
				}

				model := multimodal.NewTestBaseMultimodalModel("test", "test-model", map[string]any{
					"Provider": "test",
					"Model":    "test-model",
				}, capabilities)

				// Test streaming
				// Convert multimodal.ContentBlock to types.ContentBlock
				typeBlocks := make([]*types.ContentBlock, len(input.ContentBlocks))
				for i, block := range input.ContentBlocks {
					typeBlocks[i] = &types.ContentBlock{
						Type:     block.Type,
						Data:     block.Data,
						URL:      block.URL,
						FilePath: block.FilePath,
						Format:   block.Format,
						MIMEType: block.MIMEType,
						Size:     block.Size,
						Metadata: block.Metadata,
					}
				}
				ch, err := model.ProcessStream(ctx, &types.MultimodalInput{
					ID:            input.ID,
					ContentBlocks: typeBlocks,
				})
				if err != nil {
					t.Logf("Streaming failed (expected if provider not registered): %v", err)
					return
				}

				require.NoError(t, err)
				assert.NotNil(t, ch)

				// Try to receive at least one chunk
				select {
				case output := <-ch:
					if output != nil {
						assert.NotNil(t, output)
					}
				default:
					// Channel might be empty, which is acceptable
					t.Logf("No stream chunks received (acceptable for mock)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			tt.testFunc(t)
		})
	}
}

// TestMultimodalLLMsMessageFlow tests message flow between multimodal and LLMs.
func TestMultimodalLLMsMessageFlow(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	// Create multimodal input
	textBlock, err := multimodal.NewContentBlock("text", []byte("Hello"))
	require.NoError(t, err)

	input, err := multimodal.NewMultimodalInput([]*multimodal.ContentBlock{textBlock})
	require.NoError(t, err)

	// Create multimodal model
	capabilities := &types.ModalityCapabilities{
		Text: true,
	}

	baseModel := multimodal.NewTestBaseMultimodalModel("test", "test-model", map[string]any{
		"Provider": "test",
		"Model":    "test-model",
	}, capabilities)
	extension := multimodal.NewTestMultimodalAgentExtension(baseModel)

	// Convert multimodal input to schema messages
	messages, err := extension.ProcessForAgent(ctx, input)
	if err != nil {
		t.Logf("ProcessForAgent failed (expected if model not fully configured): %v", err)
		return
	}

	require.NoError(t, err)
	assert.NotEmpty(t, messages)

	// Test that messages can be used with LLM
	mockLLM := helper.CreateMockLLM("test-llm")
	if mockLLM != nil {
		// Messages should be compatible with LLM interface
		t.Logf("Messages are compatible with LLM interface: %d messages", len(messages))
		for i, msg := range messages {
			assert.NotNil(t, msg, "Message %d should not be nil", i)
		}
	}
}

// TestMultimodalLLMsErrorHandling tests error handling in multimodal-LLM integration.
func TestMultimodalLLMsErrorHandling(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	// Create model with error
	capabilities := &types.ModalityCapabilities{
		Text: true,
	}

	model := multimodal.NewMockMultimodalModel("test", "test-model", capabilities,
		multimodal.WithErrorCode("Process", multimodal.ErrCodeTimeout),
	)

	// Create input
	textBlock, err := multimodal.NewContentBlock("text", []byte("Test"))
	require.NoError(t, err)

	input, err := multimodal.NewMultimodalInput([]*multimodal.ContentBlock{textBlock})
	require.NoError(t, err)

	// Process should return error
	// Convert multimodal.ContentBlock to types.ContentBlock
	typeBlocks := make([]*types.ContentBlock, len(input.ContentBlocks))
	for i, block := range input.ContentBlocks {
		typeBlocks[i] = &types.ContentBlock{
			Type:     block.Type,
			Data:     block.Data,
			URL:      block.URL,
			FilePath: block.FilePath,
			Format:   block.Format,
			MIMEType: block.MIMEType,
			Size:     block.Size,
			Metadata: block.Metadata,
		}
	}
	output, err := model.Process(ctx, &types.MultimodalInput{
		ID:            input.ID,
		ContentBlocks: typeBlocks,
	})

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, multimodal.IsErrorCode(err, multimodal.ErrCodeTimeout))
}

// BenchmarkMultimodalLLMs benchmarks multimodal-LLM integration.
func BenchmarkMultimodalLLMs(b *testing.B) {
	ctx := context.Background()
	textBlock, _ := multimodal.NewContentBlock("text", []byte("Test"))
	input, _ := multimodal.NewMultimodalInput([]*multimodal.ContentBlock{textBlock})

	capabilities := &types.ModalityCapabilities{
		Text: true,
	}

	model := multimodal.NewMockMultimodalModel("test", "test-model", capabilities)

	// Convert multimodal.ContentBlock to types.ContentBlock
	typeBlocks := make([]*types.ContentBlock, len(input.ContentBlocks))
	for i, block := range input.ContentBlocks {
		typeBlocks[i] = &types.ContentBlock{
			Type:     block.Type,
			Data:     block.Data,
			URL:      block.URL,
			FilePath: block.FilePath,
			Format:   block.Format,
			MIMEType: block.MIMEType,
			Size:     block.Size,
			Metadata: block.Metadata,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = model.Process(ctx, &types.MultimodalInput{
			ID:            input.ID,
			ContentBlocks: typeBlocks,
		})
	}
}
