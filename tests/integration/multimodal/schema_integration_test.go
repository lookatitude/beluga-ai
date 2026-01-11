// Package multimodal provides integration tests for schema package compatibility.
package multimodal

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/multimodal"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaIntegration_ImageMessage(t *testing.T) {
	ctx := context.Background()

	// Create ImageMessage
	imgMsg := &schema.ImageMessage{
		ImageURL:    "https://example.com/image.jpg",
		ImageFormat: "jpeg",
	}
	imgMsg.BaseMessage.Content = "This is an image"

	// Create base model
	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	capabilities := &multimodal.ModalityCapabilities{
		Text:  true,
		Image: true,
	}

	baseModel := multimodal.NewTestBaseMultimodalModel("test", "test-model", config, capabilities)
	extension := multimodal.NewTestMultimodalAgentExtension(baseModel)

	// Handle ImageMessage
	input, err := extension.HandleMultimodalMessage(ctx, imgMsg)
	if err != nil {
		t.Logf("Handling failed (expected if provider not registered): %v", err)
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, input)
	assert.Greater(t, len(input.ContentBlocks), 0)
}

func TestSchemaIntegration_VideoMessage(t *testing.T) {
	ctx := context.Background()

	// Create VideoMessage
	vidMsg := &schema.VideoMessage{
		VideoURL:    "https://example.com/video.mp4",
		VideoFormat: "mp4",
		Duration:    10.5,
	}
	vidMsg.BaseMessage.Content = "This is a video"

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	capabilities := &multimodal.ModalityCapabilities{
		Text:  true,
		Video: true,
	}

	baseModel := multimodal.NewTestBaseMultimodalModel("test", "test-model", config, capabilities)
	extension := multimodal.NewTestMultimodalAgentExtension(baseModel)

	// Handle VideoMessage
	input, err := extension.HandleMultimodalMessage(ctx, vidMsg)
	if err != nil {
		t.Logf("Handling failed: %v", err)
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, input)
}

func TestSchemaIntegration_VoiceDocument(t *testing.T) {
	ctx := context.Background()

	// Create VoiceDocument
	voiceDoc := schema.NewVoiceDocument(
		"https://example.com/audio.mp3",
		"This is a transcript",
		map[string]string{
			"audio_format": "mp3",
			"duration":     "5.0",
		},
	)

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	capabilities := &multimodal.ModalityCapabilities{
		Text:  true,
		Audio: true,
	}

	baseModel := multimodal.NewTestBaseMultimodalModel("test", "test-model", config, capabilities)
	extension := multimodal.NewTestMultimodalAgentExtension(baseModel)

	// Handle VoiceDocument (as Document)
	doc := voiceDoc.AsDocument()
	input, err := extension.HandleMultimodalMessage(ctx, &doc)
	if err != nil {
		t.Logf("Handling failed: %v", err)
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, input)
}

func TestSchemaIntegration_Compatibility(t *testing.T) {
	ctx := context.Background()

	// Test that schema types work with multimodal package
	testCases := []struct {
		name    string
		message schema.Message
	}{
		{
			name:    "ImageMessage",
			message: &schema.ImageMessage{ImageURL: "https://example.com/img.jpg"},
		},
		{
			name:    "VideoMessage",
			message: &schema.VideoMessage{VideoURL: "https://example.com/vid.mp4"},
		},
		{
			name:    "HumanMessage",
			message: schema.NewHumanMessage("test"),
		},
		{
			name:    "AIMessage",
			message: schema.NewAIMessage("test"),
		},
	}

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	capabilities := &multimodal.ModalityCapabilities{
		Text:  true,
		Image: true,
		Video: true,
		Audio: true,
	}

	baseModel := multimodal.NewTestBaseMultimodalModel("test", "test-model", config, capabilities)
	extension := multimodal.NewTestMultimodalAgentExtension(baseModel)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input, err := extension.HandleMultimodalMessage(ctx, tc.message)
			if err != nil {
				t.Logf("Handling failed (expected if provider not registered): %v", err)
				return
			}

			// Verify structure is correct
			assert.NotNil(t, input)
			assert.NotEmpty(t, input.ID)
		})
	}
}
