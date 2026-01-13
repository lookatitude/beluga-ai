package vocode

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// VocodeProvider implements the BackendProvider interface for Vocode.
type VocodeProvider struct{}

// NewVocodeProvider creates a new Vocode provider.
func NewVocodeProvider() *VocodeProvider {
	return &VocodeProvider{}
}

// GetName returns the provider name.
func (p *VocodeProvider) GetName() string {
	return "vocode"
}

// GetCapabilities returns the capabilities of the Vocode provider.
func (p *VocodeProvider) GetCapabilities(ctx context.Context) (*iface.ProviderCapabilities, error) {
	return &iface.ProviderCapabilities{
		S2SSupport:            true,
		MultiUserSupport:      true,
		SessionPersistence:    true,
		CustomAuth:            true,
		CustomRateLimiting:    true,
		MaxConcurrentSessions: 0,                      // Vocode supports unlimited sessions
		MinLatency:            150 * time.Millisecond, // Vocode can achieve <200ms latency
		SupportedCodecs:       []string{"opus", "pcm", "g722"},
	}, nil
}

// CreateBackend creates a new Vocode backend instance.
func (p *VocodeProvider) CreateBackend(ctx context.Context, config *iface.Config) (iface.VoiceBackend, error) {
	vocodeConfig := NewVocodeConfig(config)

	// Validate Vocode-specific config
	if err := p.ValidateConfig(ctx, config); err != nil {
		return nil, err
	}

	return NewVocodeBackend(vocodeConfig)
}

// ValidateConfig validates the Vocode provider configuration.
func (p *VocodeProvider) ValidateConfig(ctx context.Context, config *iface.Config) error {
	// Validate base config
	if err := backend.ValidateConfig(config); err != nil {
		return err
	}

	// Validate Vocode-specific fields
	vocodeConfig := NewVocodeConfig(config)

	if vocodeConfig.APIKey == "" {
		return errors.New("api_key is required for Vocode provider")
	}

	if vocodeConfig.APIURL == "" {
		return errors.New("api_url is required for Vocode provider")
	}

	// Validate URL format (basic check)
	if !isValidURL(vocodeConfig.APIURL) {
		return fmt.Errorf("invalid api_url format: %s", vocodeConfig.APIURL)
	}

	return nil
}

// GetConfigSchema returns the configuration schema for the Vocode provider.
func (p *VocodeProvider) GetConfigSchema() *iface.ConfigSchema {
	return &iface.ConfigSchema{
		Fields: []iface.ConfigField{
			{
				Name:        "provider",
				Type:        "string",
				Required:    true,
				Description: "Provider name (must be 'vocode')",
			},
			{
				Name:        "api_key",
				Type:        "string",
				Required:    true,
				Description: "Vocode API key",
			},
			{
				Name:        "api_url",
				Type:        "string",
				Required:    false,
				Description: "Vocode API base URL (default: 'https://api.vocode.dev')",
				Default:     "https://api.vocode.dev",
			},
			{
				Name:        "agent_id",
				Type:        "string",
				Required:    false,
				Description: "Vocode agent ID (optional, can be created via API)",
			},
			{
				Name:        "phone_number_id",
				Type:        "string",
				Required:    false,
				Description: "Phone number ID for telephony (optional)",
			},
			{
				Name:        "enable_recording",
				Type:        "boolean",
				Required:    false,
				Description: "Enable call recording (default: false)",
				Default:     false,
			},
			{
				Name:        "recording_type",
				Type:        "string",
				Required:    false,
				Description: "Recording type if enabled ('transcript' or 'audio', default: 'transcript')",
				Default:     "transcript",
			},
			{
				Name:        "max_call_duration",
				Type:        "duration",
				Required:    false,
				Description: "Maximum call duration (default: 30m)",
				Default:     "30m",
			},
		},
	}
}

// isValidURL performs basic URL validation.
func isValidURL(url string) bool {
	return len(url) > 0 && (url[:4] == "http" || url[:5] == "https")
}
