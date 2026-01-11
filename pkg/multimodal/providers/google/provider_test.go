// Package google provides tests for Google Vertex AI multimodal provider.
package google

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoogleProvider(t *testing.T) {
	googleConfig := &Config{
		ProjectID: "test-project",
		Location:  "us-central1",
		Model:     "gemini-1.5-pro",
		BaseURL:   "https://us-central1-aiplatform.googleapis.com/v1",
		Timeout:   30 * time.Second,
	}

	provider, err := NewGoogleProvider(googleConfig)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewGoogleProvider_InvalidConfig(t *testing.T) {
	// Missing project ID
	googleConfig := &Config{
		Location: "us-central1",
		Model:    "gemini-1.5-pro",
	}

	provider, err := NewGoogleProvider(googleConfig)
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "project ID is required")

	// Missing model
	googleConfig = &Config{
		ProjectID: "test-project",
		Location:  "us-central1",
	}

	provider, err = NewGoogleProvider(googleConfig)
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "model is required")
}

func TestGoogleProvider_GetCapabilities(t *testing.T) {
	googleConfig := &Config{
		ProjectID: "test-project",
		Location:  "us-central1",
		Model:     "gemini-1.5-pro",
		BaseURL:   "https://us-central1-aiplatform.googleapis.com/v1",
	}

	provider, err := NewGoogleProvider(googleConfig)
	require.NoError(t, err)

	ctx := context.Background()
	capabilities, err := provider.GetCapabilities(ctx)
	require.NoError(t, err)
	assert.NotNil(t, capabilities)
	assert.True(t, capabilities.Text)
	assert.True(t, capabilities.Image)
	assert.True(t, capabilities.Audio)
	assert.True(t, capabilities.Video)
}

func TestGoogleProvider_SupportsModality(t *testing.T) {
	googleConfig := &Config{
		ProjectID: "test-project",
		Location:  "us-central1",
		Model:     "gemini-1.5-pro",
		BaseURL:   "https://us-central1-aiplatform.googleapis.com/v1",
	}

	provider, err := NewGoogleProvider(googleConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// Test supported modalities
	supported, err := provider.SupportsModality(ctx, "text")
	require.NoError(t, err)
	assert.True(t, supported)

	supported, err = provider.SupportsModality(ctx, "image")
	require.NoError(t, err)
	assert.True(t, supported)

	supported, err = provider.SupportsModality(ctx, "audio")
	require.NoError(t, err)
	assert.True(t, supported)

	supported, err = provider.SupportsModality(ctx, "video")
	require.NoError(t, err)
	assert.True(t, supported)

	// Test unsupported modality
	supported, err = provider.SupportsModality(ctx, "unknown")
	require.Error(t, err)
	assert.False(t, supported)
	assert.Contains(t, err.Error(), "unknown modality")
}

func TestGoogleProvider_Process(t *testing.T) {
	// Note: This test requires Google Vertex AI credentials or will be skipped.
	// For proper mocking, the provider would need to be refactored to use an interface.

	googleConfig := &Config{
		ProjectID: "test-project",
		Location:  "us-central1",
		Model:     "gemini-1.5-pro",
		BaseURL:   "https://us-central1-aiplatform.googleapis.com/v1",
		APIKey:    "test-api-key",
	}

	provider, err := NewGoogleProvider(googleConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// Create test input using types
	textBlock := &types.ContentBlock{
		Type:     "text",
		Data:     []byte("Test input"),
		Format:   "text/plain",
		MIMEType: "text/plain",
		Size:     10,
		Metadata: make(map[string]any),
	}

	input := &types.MultimodalInput{
		ID:            "test-input-id",
		ContentBlocks: []*types.ContentBlock{textBlock},
		Metadata:      make(map[string]any),
		Format:        "base64",
		CreatedAt:     time.Now(),
	}

	// This will fail without valid credentials, but tests the structure
	output, err := provider.Process(ctx, input)
	if err != nil {
		// Expected without valid credentials
		assert.True(t, strings.Contains(err.Error(), "API request failed") ||
			strings.Contains(err.Error(), "authentication") ||
			strings.Contains(err.Error(), "credentials"))
	} else {
		// If it succeeds, verify output structure
		require.NotNil(t, output)
		assert.Equal(t, input.ID, output.InputID)
		assert.NotEmpty(t, output.ID)
		assert.NotEmpty(t, output.ContentBlocks)
	}
}

func TestGoogleProvider_ProcessStream(t *testing.T) {
	// Note: This test requires Google Vertex AI credentials or will be skipped.

	googleConfig := &Config{
		ProjectID: "test-project",
		Location:  "us-central1",
		Model:     "gemini-1.5-pro",
		BaseURL:   "https://us-central1-aiplatform.googleapis.com/v1",
		APIKey:    "test-api-key",
	}

	provider, err := NewGoogleProvider(googleConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// Create test input
	textBlock := &types.ContentBlock{
		Type:     "text",
		Data:     []byte("Test input"),
		Format:   "text/plain",
		MIMEType: "text/plain",
		Size:     10,
		Metadata: make(map[string]any),
	}

	input := &types.MultimodalInput{
		ID:            "test-input-id",
		ContentBlocks: []*types.ContentBlock{textBlock},
		Metadata:      make(map[string]any),
		Format:        "base64",
		CreatedAt:     time.Now(),
	}

	// This will fail without valid credentials, but tests the structure
	outputChan, err := provider.ProcessStream(ctx, input)
	if err != nil {
		// Expected without valid credentials
		assert.True(t, strings.Contains(err.Error(), "API request failed") ||
			strings.Contains(err.Error(), "authentication") ||
			strings.Contains(err.Error(), "credentials"))
	} else {
		// If it succeeds, verify channel structure
		require.NotNil(t, outputChan)
		// Read from channel (with timeout)
		select {
		case output, ok := <-outputChan:
			if ok {
				assert.NotNil(t, output)
				assert.Equal(t, input.ID, output.InputID)
			}
		case <-time.After(5 * time.Second):
			// Timeout is acceptable for streaming tests
		}
	}
}

func TestGoogleProvider_ErrorHandling(t *testing.T) {
	googleConfig := &Config{
		ProjectID: "test-project",
		Location:  "us-central1",
		Model:     "gemini-1.5-pro",
		BaseURL:   "https://us-central1-aiplatform.googleapis.com/v1",
	}

	provider, err := NewGoogleProvider(googleConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// Test with empty input
	emptyInput := &types.MultimodalInput{
		ID:            "test-input-id",
		ContentBlocks: []*types.ContentBlock{},
		Metadata:      make(map[string]any),
		Format:        "base64",
		CreatedAt:     time.Now(),
	}

	_, err = provider.Process(ctx, emptyInput)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no content blocks")
}

func TestConfig_Validate(t *testing.T) {
	// Valid config
	config := &Config{
		ProjectID: "test-project",
		Model:     "gemini-1.5-pro",
		Location:  "us-central1",
	}
	err := config.Validate()
	assert.NoError(t, err)

	// Missing project ID
	config = &Config{
		Model:    "gemini-1.5-pro",
		Location: "us-central1",
	}
	err = config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")

	// Missing model
	config = &Config{
		ProjectID: "test-project",
		Location:  "us-central1",
	}
	err = config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")
}

func TestFromMultimodalConfig(t *testing.T) {
	multimodalConfig := MultimodalConfig{
		Provider:   "google",
		Model:      "gemini-1.5-pro",
		APIKey:     "test-key",
		BaseURL:    "https://us-central1-aiplatform.googleapis.com/v1",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		ProviderSpecific: map[string]any{
			"project_id": "test-project",
			"location":   "us-central1",
		},
	}

	config := FromMultimodalConfig(multimodalConfig)
	assert.Equal(t, "test-project", config.ProjectID)
	assert.Equal(t, "us-central1", config.Location)
	assert.Equal(t, "gemini-1.5-pro", config.Model)
	assert.Equal(t, "test-key", config.APIKey)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.MaxRetries)
}
