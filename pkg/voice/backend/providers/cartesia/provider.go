package cartesia

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// CartesiaProvider implements the BackendProvider interface for Cartesia.
type CartesiaProvider struct{}

// NewCartesiaProvider creates a new Cartesia provider.
func NewCartesiaProvider() *CartesiaProvider {
	return &CartesiaProvider{}
}

// GetName returns the provider name.
func (p *CartesiaProvider) GetName() string {
	return "cartesia"
}

// GetCapabilities returns the capabilities of the Cartesia provider.
func (p *CartesiaProvider) GetCapabilities(ctx context.Context) (*iface.ProviderCapabilities, error) {
	return &iface.ProviderCapabilities{
		S2SSupport:            true,
		MultiUserSupport:      true,
		SessionPersistence:    true,
		CustomAuth:            true,
		CustomRateLimiting:    true,
		MaxConcurrentSessions: 0, // Cartesia supports unlimited sessions
		MinLatency:            100 * time.Millisecond, // Cartesia can achieve <150ms latency
		SupportedCodecs:       []string{"opus", "pcm", "g722"},
	}, nil
}

// CreateBackend creates a new Cartesia backend instance.
func (p *CartesiaProvider) CreateBackend(ctx context.Context, config *iface.Config) (iface.VoiceBackend, error) {
	cartesiaConfig := NewCartesiaConfig(config)

	// Validate Cartesia-specific config
	if err := p.ValidateConfig(ctx, config); err != nil {
		return nil, err
	}

	return NewCartesiaBackend(cartesiaConfig)
}

// ValidateConfig validates the Cartesia provider configuration.
func (p *CartesiaProvider) ValidateConfig(ctx context.Context, config *iface.Config) error {
	// Validate base config
	if err := backend.ValidateConfig(config); err != nil {
		return err
	}

	// Validate Cartesia-specific fields
	cartesiaConfig := NewCartesiaConfig(config)

	if cartesiaConfig.APIKey == "" {
		return errors.New("api_key is required for Cartesia provider")
	}

	if cartesiaConfig.APIURL == "" {
		return errors.New("api_url is required for Cartesia provider")
	}

	// Validate URL format (basic check)
	if !isValidURL(cartesiaConfig.APIURL) {
		return fmt.Errorf("invalid api_url format: %s", cartesiaConfig.APIURL)
	}

	return nil
}

// GetConfigSchema returns the configuration schema for the Cartesia provider.
func (p *CartesiaProvider) GetConfigSchema() *iface.ConfigSchema {
	return &iface.ConfigSchema{
		Fields: []iface.ConfigField{
			{
				Name:        "provider",
				Type:        "string",
				Required:    true,
				Description: "Provider name (must be 'cartesia')",
			},
			{
				Name:        "api_key",
				Type:        "string",
				Required:    true,
				Description: "Cartesia API key",
			},
			{
				Name:        "api_url",
				Type:        "string",
				Required:    false,
				Description: "Cartesia API base URL (default: 'https://api.cartesia.ai')",
				Default:     "https://api.cartesia.ai",
			},
			{
				Name:        "model_id",
				Type:        "string",
				Required:    false,
				Description: "Cartesia model ID for voice generation (optional)",
			},
			{
				Name:        "voice_id",
				Type:        "string",
				Required:    false,
				Description: "Cartesia voice ID (optional)",
			},
			{
				Name:        "enable_streaming",
				Type:        "boolean",
				Required:    false,
				Description: "Enable streaming audio generation (default: true)",
				Default:     true,
			},
			{
				Name:        "sample_rate",
				Type:        "integer",
				Required:    false,
				Description: "Audio sample rate (16000, 24000, or 48000, default: 24000)",
				Default:     24000,
			},
			{
				Name:        "latency_optimization",
				Type:        "boolean",
				Required:    false,
				Description: "Enable latency optimization (default: true)",
				Default:     true,
			},
		},
	}
}

// isValidURL performs basic URL validation.
func isValidURL(url string) bool {
	return len(url) > 0 && (url[:4] == "http" || url[:5] == "https")
}
