package anthropic

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
)

func TestNewAnthropicProvider(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping test")
	}

	config := &Config{
		APIKey:     apiKey,
		Model:      "claude-3-5-sonnet-20241022",
		BaseURL:    "",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		Enabled:    true,
	}

	provider, err := NewAnthropicProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Anthropic provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}
}

func TestAnthropicProvider_GetCapabilities(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping test")
	}

	config := &Config{
		APIKey:     apiKey,
		Model:      "claude-3-5-sonnet-20241022",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	provider, err := NewAnthropicProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Anthropic provider: %v", err)
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

func TestAnthropicProvider_SupportsModality(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping test")
	}

	config := &Config{
		APIKey:     apiKey,
		Model:      "claude-3-5-sonnet-20241022",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	provider, err := NewAnthropicProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Anthropic provider: %v", err)
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

func TestAnthropicProvider_Process(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" || apiKey == "test-api-key" {
		t.Skip("ANTHROPIC_API_KEY not set or is placeholder, skipping test")
	}

	config := &Config{
		APIKey:     apiKey,
		Model:      "claude-3-5-sonnet-20241022",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	provider, err := NewAnthropicProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Anthropic provider: %v", err)
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
				Model:  "claude-3-5-sonnet-20241022",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &Config{
				Model: "claude-3-5-sonnet-20241022",
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
		Provider:   "anthropic",
		Model:      "claude-3-5-sonnet-20241022",
		APIKey:     "test-key",
		BaseURL:    "https://api.anthropic.com/v1",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		ProviderSpecific: map[string]any{
			"api_version": "2023-06-01",
		},
	}

	config := FromMultimodalConfig(multimodalConfig)

	if config.APIKey != "test-key" {
		t.Errorf("Expected APIKey to be 'test-key', got %s", config.APIKey)
	}
	if config.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected Model to be 'claude-3-5-sonnet-20241022', got %s", config.Model)
	}
	if config.APIVersion != "2023-06-01" {
		t.Errorf("Expected APIVersion to be '2023-06-01', got %s", config.APIVersion)
	}
}
