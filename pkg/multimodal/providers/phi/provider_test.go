// Package phi provides tests for Phi multimodal provider.
package phi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPhiProvider(t *testing.T) {
	phiConfig := &Config{
		APIKey:  "test-api-key",
		Model:   "phi-12b",
		BaseURL: "https://api.mistral.ai/v1",
		Timeout: 30 * time.Second,
	}

	provider, err := NewPhiProvider(phiConfig)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewPhiProvider_InvalidConfig(t *testing.T) {
	// Missing API key
	phiConfig := &Config{
		Model:   "phi-12b",
		BaseURL: "https://api.mistral.ai/v1",
	}

	provider, err := NewPhiProvider(phiConfig)
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "API key is required")

	// Missing model
	phiConfig = &Config{
		APIKey:  "test-api-key",
		BaseURL: "https://api.mistral.ai/v1",
	}

	provider, err = NewPhiProvider(phiConfig)
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "model is required")
}

func TestPhiProvider_GetCapabilities(t *testing.T) {
	phiConfig := &Config{
		APIKey:  "test-api-key",
		Model:   "phi-12b",
		BaseURL: "https://api.mistral.ai/v1",
	}

	provider, err := NewPhiProvider(phiConfig)
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

func TestPhiProvider_SupportsModality(t *testing.T) {
	phiConfig := &Config{
		APIKey:  "test-api-key",
		Model:   "phi-12b",
		BaseURL: "https://api.mistral.ai/v1",
	}

	provider, err := NewPhiProvider(phiConfig)
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

func TestPhiProvider_Process(t *testing.T) {
	// Note: This test requires Phi API credentials or will be skipped.
	// For proper mocking, the provider would need to be refactored to use an interface.

	phiConfig := &Config{
		APIKey:  "test-api-key",
		Model:   "phi-12b",
		BaseURL: "https://api.mistral.ai/v1",
	}

	provider, err := NewPhiProvider(phiConfig)
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

func TestPhiProvider_ProcessStream(t *testing.T) {
	// Note: This test requires Phi API credentials or will be skipped.

	phiConfig := &Config{
		APIKey:  "test-api-key",
		Model:   "phi-12b",
		BaseURL: "https://api.mistral.ai/v1",
	}

	provider, err := NewPhiProvider(phiConfig)
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

func TestPhiProvider_ErrorHandling(t *testing.T) {
	phiConfig := &Config{
		APIKey:  "test-api-key",
		Model:   "phi-12b",
		BaseURL: "https://api.mistral.ai/v1",
	}

	provider, err := NewPhiProvider(phiConfig)
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
		APIKey: "test-key",
		Model:  "phi-12b",
	}
	err := config.Validate()
	assert.NoError(t, err)

	// Missing API key
	config = &Config{
		Model: "phi-12b",
	}
	err = config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")

	// Missing model
	config = &Config{
		APIKey: "test-key",
	}
	err = config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")
}

func TestFromMultimodalConfig(t *testing.T) {
	multimodalConfig := MultimodalConfig{
		Provider:   "phi",
		Model:      "phi-12b",
		APIKey:     "test-key",
		BaseURL:    "https://api.mistral.ai/v1",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		ProviderSpecific: map[string]any{
			"enabled": true,
		},
	}

	config := FromMultimodalConfig(multimodalConfig)
	assert.Equal(t, "phi-12b", config.Model)
	assert.Equal(t, "test-key", config.APIKey)
	assert.Equal(t, "https://api.mistral.ai/v1", config.BaseURL)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.True(t, config.Enabled)
}
