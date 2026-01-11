package xai

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
)

func TestNewXAIProvider(t *testing.T) {
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set, skipping test")
	}

	config := &Config{
		APIKey:     apiKey,
		Model:      "grok-beta",
		BaseURL:    "https://api.x.ai/v1",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		Enabled:    true,
	}

	provider, err := NewXAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create xAI provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}
}

func TestXAIProvider_GetCapabilities(t *testing.T) {
	config := &Config{
		APIKey:  "test-api-key",
		Model:   "grok-beta",
		BaseURL: "https://api.x.ai/v1",
	}

	provider, err := NewXAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create xAI provider: %v", err)
	}

	ctx := context.Background()
	capabilities, err := provider.GetCapabilities(ctx)
	if err != nil {
		t.Fatalf("Failed to get capabilities: %v", err)
	}

	if !capabilities.Text {
		t.Error("Expected text capability to be true")
	}
	if !capabilities.Image {
		t.Error("Expected image capability to be true")
	}
}

func TestXAIProvider_SupportsModality(t *testing.T) {
	config := &Config{
		APIKey:  "test-api-key",
		Model:   "grok-beta",
		BaseURL: "https://api.x.ai/v1",
	}

	provider, err := NewXAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create xAI provider: %v", err)
	}

	ctx := context.Background()

	modalities := []string{"text", "image", "audio", "video"}
	for _, modality := range modalities {
		supported, err := provider.SupportsModality(ctx, modality)
		if err != nil {
			t.Errorf("Failed to check modality %s: %v", modality, err)
		}
		if modality == "text" && !supported {
			t.Errorf("Expected %s to be supported", modality)
		}
	}
}

func TestXAIProvider_Process(t *testing.T) {
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" || apiKey == "test-api-key" {
		t.Skip("XAI_API_KEY not set or is placeholder, skipping test")
	}

	config := &Config{
		APIKey:     apiKey,
		Model:      "grok-beta",
		BaseURL:    "https://api.x.ai/v1",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	provider, err := NewXAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create xAI provider: %v", err)
	}

	ctx := context.Background()

	input := &types.MultimodalInput{
		ID: "test-input-1",
		ContentBlocks: []*types.ContentBlock{
			{
				Type:     "text",
				Data:     []byte("What is 2+2?"),
				Format:   "text/plain",
				MIMEType: "text/plain",
				Size:     10,
			},
		},
	}

	output, err := provider.Process(ctx, input)
	if err != nil {
		// Check if it's an API key error and skip gracefully
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "api key") || strings.Contains(errStr, "authentication") || strings.Contains(errStr, "401") || strings.Contains(errStr, "invalid") {
			t.Skipf("Skipping test due to invalid API key: %v", err)
		}
		t.Fatalf("Failed to process input: %v", err)
	}

	if output == nil {
		t.Fatal("Output is nil")
	}

	if len(output.ContentBlocks) == 0 {
		t.Error("Expected at least one content block in output")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				APIKey: "test-key",
				Model:  "grok-beta",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &Config{
				Model: "grok-beta",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFromMultimodalConfig(t *testing.T) {
	multimodalConfig := MultimodalConfig{
		Provider:   "xai",
		Model:      "grok-beta",
		APIKey:     "test-key",
		BaseURL:    "https://api.x.ai/v1",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		ProviderSpecific: map[string]any{
			"enabled": true,
		},
	}

	config := FromMultimodalConfig(multimodalConfig)

	if config.APIKey != "test-key" {
		t.Errorf("Expected APIKey to be 'test-key', got %s", config.APIKey)
	}
	if config.Model != "grok-beta" {
		t.Errorf("Expected Model to be 'grok-beta', got %s", config.Model)
	}
}
