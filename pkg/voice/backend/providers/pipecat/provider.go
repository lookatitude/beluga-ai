package pipecat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// PipecatProvider implements the BackendProvider interface for Pipecat (via Daily.co).
type PipecatProvider struct{}

// NewPipecatProvider creates a new Pipecat provider.
func NewPipecatProvider() *PipecatProvider {
	return &PipecatProvider{}
}

// GetName returns the provider name.
func (p *PipecatProvider) GetName() string {
	return "pipecat"
}

// GetCapabilities returns the capabilities of the Pipecat provider.
func (p *PipecatProvider) GetCapabilities(ctx context.Context) (*iface.ProviderCapabilities, error) {
	return &iface.ProviderCapabilities{
		S2SSupport:            true,
		MultiUserSupport:      true,
		SessionPersistence:    true,
		CustomAuth:            true,
		CustomRateLimiting:    true,
		MaxConcurrentSessions: 0, // Daily.co supports unlimited sessions
		MinLatency:            100 * time.Millisecond, // Daily.co can achieve <200ms latency
		SupportedCodecs:       []string{"opus", "pcm", "g722"},
	}, nil
}

// CreateBackend creates a new Pipecat backend instance.
func (p *PipecatProvider) CreateBackend(ctx context.Context, config *iface.Config) (iface.VoiceBackend, error) {
	pipecatConfig := NewPipecatConfig(config)

	// Validate Pipecat-specific config
	if err := p.ValidateConfig(ctx, config); err != nil {
		return nil, err
	}

	return NewPipecatBackend(pipecatConfig)
}

// ValidateConfig validates the Pipecat provider configuration.
func (p *PipecatProvider) ValidateConfig(ctx context.Context, config *iface.Config) error {
	// Validate base config
	if err := backend.ValidateConfig(config); err != nil {
		return err
	}

	// Validate Pipecat-specific fields
	pipecatConfig := NewPipecatConfig(config)

	if pipecatConfig.DailyAPIKey == "" {
		return errors.New("daily_api_key is required for Pipecat provider")
	}

	if pipecatConfig.DailyAPIURL == "" {
		return errors.New("daily_api_url is required for Pipecat provider")
	}

	// Validate URL format (basic check)
	if !isValidURL(pipecatConfig.DailyAPIURL) {
		return fmt.Errorf("invalid daily_api_url format: %s", pipecatConfig.DailyAPIURL)
	}

	return nil
}

// GetConfigSchema returns the configuration schema for the Pipecat provider.
func (p *PipecatProvider) GetConfigSchema() *iface.ConfigSchema {
	return &iface.ConfigSchema{
		Fields: []iface.ConfigField{
			{
				Name:        "provider",
				Type:        "string",
				Required:    true,
				Description: "Provider name (must be 'pipecat')",
			},
			{
				Name:        "daily_api_key",
				Type:        "string",
				Required:    true,
				Description: "Daily.co API key",
			},
			{
				Name:        "daily_api_url",
				Type:        "string",
				Required:    false,
				Description: "Daily.co API base URL (default: 'https://api.daily.co/v1')",
				Default:     "https://api.daily.co/v1",
			},
			{
				Name:        "pipecat_server_url",
				Type:        "string",
				Required:    false,
				Description: "Pipecat server URL (optional, if using Pipecat server)",
			},
			{
				Name:        "room_name_prefix",
				Type:        "string",
				Required:    false,
				Description: "Prefix for Daily.co room names (default: 'beluga-')",
				Default:     "beluga-",
			},
			{
				Name:        "enable_recording",
				Type:        "boolean",
				Required:    false,
				Description: "Enable room recording (default: false)",
				Default:     false,
			},
			{
				Name:        "recording_type",
				Type:        "string",
				Required:    false,
				Description: "Recording type if enabled ('cloud' or 'local', default: 'cloud')",
				Default:     "cloud",
			},
			{
				Name:        "max_participants",
				Type:        "integer",
				Required:    false,
				Description: "Maximum participants per room (default: unlimited)",
				Default:     0,
			},
		},
	}
}

// isValidURL performs basic URL validation.
func isValidURL(url string) bool {
	return len(url) > 0 && (url[:4] == "http" || url[:5] == "https")
}
