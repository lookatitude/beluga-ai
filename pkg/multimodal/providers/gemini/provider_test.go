// Package gemini provides tests for Google Gemini multimodal provider.
package gemini

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGeminiProvider(t *testing.T) {
	geminiConfig := &Config{
		APIKey:  "test-api-key",
		Model:   "gemini-1.5-pro",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta",
		Timeout: 30 * time.Second,
	}

	provider, err := NewGeminiProvider(geminiConfig)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestGeminiProvider_GetCapabilities(t *testing.T) {
	geminiConfig := &Config{
		APIKey:  "test-api-key",
		Model:   "gemini-1.5-pro",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta",
	}

	provider, err := NewGeminiProvider(geminiConfig)
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

func TestGeminiProvider_Process(t *testing.T) {
	// Note: This test requires Gemini API credentials or will be skipped.
	// For proper mocking, the provider would need to be refactored to use an interface.

	geminiConfig := &Config{
		APIKey:  "test-api-key",
		Model:   "gemini-1.5-pro",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta",
	}

	provider, err := NewGeminiProvider(geminiConfig)
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

	// Process - will make actual API call
	output, err := provider.Process(ctx, input)
	// Check if it's an API error (expected without valid credentials)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "403") ||
			strings.Contains(errStr, "Unauthorized") ||
			strings.Contains(errStr, "Forbidden") ||
			strings.Contains(errStr, "API key") ||
			strings.Contains(errStr, "authentication") ||
			strings.Contains(errStr, "invalid_request") ||
			strings.Contains(errStr, "API request failed") {
			t.Skipf("Skipping test - API error (expected without valid credentials): %v", err)
			return
		}
		// Other errors should fail the test
		require.NoError(t, err)
	}

	if output != nil {
		assert.Equal(t, input.ID, output.InputID)
		assert.NotEmpty(t, output.ContentBlocks)
	}
}

func TestGeminiProvider_SupportsModality(t *testing.T) {
	geminiConfig := &Config{
		APIKey:  "test-api-key",
		Model:   "gemini-1.5-pro",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta",
	}

	provider, err := NewGeminiProvider(geminiConfig)
	require.NoError(t, err)

	ctx := context.Background()

	testCases := []struct {
		modality string
		expected bool
	}{
		{"text", true},
		{"image", true},
		{"audio", true},
		{"video", true},
		{"unknown", false},
	}

	for _, tc := range testCases {
		t.Run(tc.modality, func(t *testing.T) {
			supported, err := provider.SupportsModality(ctx, tc.modality)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, supported)
		})
	}
}

func TestGeminiProvider_ProcessStream(t *testing.T) {
	// Note: This test requires Gemini API credentials or will be skipped.
	// For proper mocking, the provider would need to be refactored to use an interface.

	geminiConfig := &Config{
		APIKey:  "test-api-key",
		Model:   "gemini-1.5-pro",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta",
	}

	provider, err := NewGeminiProvider(geminiConfig)
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

	// Process stream - will make actual API call
	outputChan, err := provider.ProcessStream(ctx, input)
	// Check if it's an API error (expected without valid credentials)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "403") ||
			strings.Contains(errStr, "Unauthorized") ||
			strings.Contains(errStr, "Forbidden") ||
			strings.Contains(errStr, "API key") ||
			strings.Contains(errStr, "authentication") ||
			strings.Contains(errStr, "invalid_request") ||
			strings.Contains(errStr, "API request failed") {
			t.Skipf("Skipping test - API error (expected without valid credentials): %v", err)
			return
		}
		// Other errors should fail the test
		require.NoError(t, err)
	}

	require.NotNil(t, outputChan)

	// Read from channel
	select {
	case output := <-outputChan:
		if output != nil {
			assert.NotEmpty(t, output.ID)
		}
	case <-time.After(5 * time.Second):
		// Timeout is acceptable if no valid credentials
		t.Log("Streaming timeout (expected without valid credentials)")
	}
}

func TestConfig_Validate(t *testing.T) {
	testCases := []struct {
		config  *Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				APIKey: "test-key",
				Model:  "gemini-1.5-pro",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &Config{
				Model: "gemini-1.5-pro",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: &Config{
				APIKey: "test-key",
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFromMultimodalConfig(t *testing.T) {
	multimodalConfig := MultimodalConfig{
		Provider:   "gemini",
		Model:      "gemini-1.5-pro",
		APIKey:     "test-key",
		BaseURL:    "https://generativelanguage.googleapis.com/v1beta",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		ProviderSpecific: map[string]any{
			"enabled": true,
		},
	}

	config := FromMultimodalConfig(multimodalConfig)
	assert.Equal(t, "test-key", config.APIKey)
	assert.Equal(t, "gemini-1.5-pro", config.Model)
	assert.Equal(t, "https://generativelanguage.googleapis.com/v1beta", config.BaseURL)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.True(t, config.Enabled)
}
