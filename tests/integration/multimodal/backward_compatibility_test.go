// Package multimodal provides integration tests for backward compatibility with text-only workflows.
package multimodal

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/multimodal"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackwardCompatibility_TextOnlyInput verifies that multimodal models
// can process text-only inputs exactly like traditional LLM workflows.
func TestBackwardCompatibility_TextOnlyInput(t *testing.T) {
	ctx := context.Background()

	// Create a mock multimodal model (text-only capable)
	model := multimodal.NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text:  true,
		Image: false,
		Audio: false,
		Video: false,
	})

	// Create text-only input (simulating traditional LLM usage)
	textBlock, err := multimodal.NewContentBlock("text", []byte("What is artificial intelligence?"))
	require.NoError(t, err)

	// Convert to types.MultimodalInput for the interface
	input := &types.MultimodalInput{
		ID: "test-input",
		ContentBlocks: []*types.ContentBlock{
			{
				Type: textBlock.Type,
				Data: textBlock.Data,
				Size: textBlock.Size,
			},
		},
	}

	// Process the text-only input
	output, err := model.Process(ctx, input)
	assert.NoError(t, err, "Text-only input should process successfully")
	assert.NotNil(t, output, "Output should not be nil")
	assert.NotEmpty(t, output.ContentBlocks, "Output should contain content blocks")

	// Verify output contains text
	if len(output.ContentBlocks) > 0 {
		assert.Equal(t, "text", output.ContentBlocks[0].Type, "Output should be text type")
	}
}

// TestBackwardCompatibility_TextOnlyStreaming verifies that streaming
// works with text-only inputs just like traditional LLM streaming.
func TestBackwardCompatibility_TextOnlyStreaming(t *testing.T) {
	ctx := context.Background()

	model := multimodal.NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text: true,
	})

	textBlock, err := multimodal.NewContentBlock("text", []byte("Explain quantum computing"))
	require.NoError(t, err)

	// Convert to types.MultimodalInput
	input := &types.MultimodalInput{
		ID: "test-input",
		ContentBlocks: []*types.ContentBlock{
			{
				Type: textBlock.Type,
				Data: textBlock.Data,
				Size: textBlock.Size,
			},
		},
	}

	// Stream the text-only input
	outputChan, err := model.ProcessStream(ctx, input)
	require.NoError(t, err, "Streaming should start successfully")

	// Receive outputs
	outputCount := 0
	for output := range outputChan {
		assert.NotNil(t, output, "Stream output should not be nil")
		outputCount++
	}

	assert.Greater(t, outputCount, 0, "Should receive at least one stream output")
}

// TestBackwardCompatibility_TextOnlyCapabilities verifies that text-only
// models report capabilities correctly and don't break existing workflows.
func TestBackwardCompatibility_TextOnlyCapabilities(t *testing.T) {
	ctx := context.Background()

	model := multimodal.NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text:  true,
		Image: false,
		Audio: false,
		Video: false,
	})

	// Check capabilities
	caps, err := model.GetCapabilities(ctx)
	require.NoError(t, err)
	assert.True(t, caps.Text, "Text capability should be true")
	assert.False(t, caps.Image, "Image capability should be false for text-only model")
	assert.False(t, caps.Audio, "Audio capability should be false")
	assert.False(t, caps.Video, "Video capability should be false")

	// Verify modality support
	supportsText, err := model.SupportsModality(ctx, "text")
	require.NoError(t, err)
	assert.True(t, supportsText, "Should support text modality")

	supportsImage, err := model.SupportsModality(ctx, "image")
	require.NoError(t, err)
	assert.False(t, supportsImage, "Should not support image modality")
}

// TestBackwardCompatibility_TextOnlyHealthCheck verifies that health checks
// work with text-only models just like with any other model.
func TestBackwardCompatibility_TextOnlyHealthCheck(t *testing.T) {
	ctx := context.Background()

	model := multimodal.NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text: true,
	})

	// Health check should work
	err := model.CheckHealth(ctx)
	assert.NoError(t, err, "Health check should pass for text-only model")
}

// TestBackwardCompatibility_MultipleTextBlocks verifies that multiple
// text blocks work correctly (simulating multi-turn conversations).
func TestBackwardCompatibility_MultipleTextBlocks(t *testing.T) {
	ctx := context.Background()

	model := multimodal.NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text: true,
	})

	// Create multiple text blocks (simulating conversation history)
	input := &types.MultimodalInput{
		ID: "test-input",
		ContentBlocks: []*types.ContentBlock{
			{Type: "text", Data: []byte("Hello"), Size: 5},
			{Type: "text", Data: []byte("How are you?"), Size: 12},
			{Type: "text", Data: []byte("What can you help me with?"), Size: 26},
		},
	}

	// Process multiple text blocks
	output, err := model.Process(ctx, input)
	assert.NoError(t, err, "Multiple text blocks should process successfully")
	assert.NotNil(t, output, "Output should not be nil")
}

// TestBackwardCompatibility_TextOnlyWithMetadata verifies that metadata
// handling works correctly with text-only inputs.
func TestBackwardCompatibility_TextOnlyWithMetadata(t *testing.T) {
	ctx := context.Background()

	model := multimodal.NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text: true,
	})

	textBlock, err := multimodal.NewContentBlock("text", []byte("Test message"))
	require.NoError(t, err)

	// Convert to types.MultimodalInput with metadata
	input := &types.MultimodalInput{
		ID: "test-input",
		ContentBlocks: []*types.ContentBlock{
			{
				Type: textBlock.Type,
				Data: textBlock.Data,
				Size: textBlock.Size,
			},
		},
		Metadata: map[string]any{
			"conversation_id": "test-123",
			"user_id":         "user-456",
		},
	}

	// Process with metadata
	output, err := model.Process(ctx, input)
	assert.NoError(t, err, "Text input with metadata should process successfully")
	assert.NotNil(t, output, "Output should not be nil")
	assert.NotNil(t, output.Metadata, "Output should have metadata")
}
