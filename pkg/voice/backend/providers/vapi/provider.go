package vapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// VapiProvider implements the BackendProvider interface for Vapi.
type VapiProvider struct{}

// NewVapiProvider creates a new Vapi provider.
func NewVapiProvider() *VapiProvider {
	return &VapiProvider{}
}

// GetName returns the provider name.
func (p *VapiProvider) GetName() string {
	return "vapi"
}

// GetCapabilities returns the capabilities of the Vapi provider.
func (p *VapiProvider) GetCapabilities(ctx context.Context) (*iface.ProviderCapabilities, error) {
	return &iface.ProviderCapabilities{
		S2SSupport:            true,
		MultiUserSupport:      true,
		SessionPersistence:    true,
		CustomAuth:            true,
		CustomRateLimiting:    true,
		MaxConcurrentSessions: 0,                      // Vapi supports unlimited sessions
		MinLatency:            150 * time.Millisecond, // Vapi can achieve <200ms latency
		SupportedCodecs:       []string{"opus", "pcm", "g722"},
	}, nil
}

// CreateBackend creates a new Vapi backend instance.
func (p *VapiProvider) CreateBackend(ctx context.Context, config *iface.Config) (iface.VoiceBackend, error) {
	vapiConfig := NewVapiConfig(config)

	// Validate Vapi-specific config
	if err := p.ValidateConfig(ctx, config); err != nil {
		return nil, err
	}

	return NewVapiBackend(vapiConfig)
}

// ValidateConfig validates the Vapi provider configuration.
func (p *VapiProvider) ValidateConfig(ctx context.Context, config *iface.Config) error {
	// Validate base config
	if err := backend.ValidateConfig(config); err != nil {
		return err
	}

	// Validate Vapi-specific fields
	vapiConfig := NewVapiConfig(config)

	if vapiConfig.APIKey == "" {
		return errors.New("api_key is required for Vapi provider")
	}

	if vapiConfig.APIURL == "" {
		return errors.New("api_url is required for Vapi provider")
	}

	// Validate URL format (basic check)
	if !isValidURL(vapiConfig.APIURL) {
		return fmt.Errorf("invalid api_url format: %s", vapiConfig.APIURL)
	}

	return nil
}

// GetConfigSchema returns the configuration schema for the Vapi provider.
func (p *VapiProvider) GetConfigSchema() *iface.ConfigSchema {
	return &iface.ConfigSchema{
		Fields: []iface.ConfigField{
			{
				Name:        "provider",
				Type:        "string",
				Required:    true,
				Description: "Provider name (must be 'vapi')",
			},
			{
				Name:        "api_key",
				Type:        "string",
				Required:    true,
				Description: "Vapi API key",
			},
			{
				Name:        "api_url",
				Type:        "string",
				Required:    false,
				Description: "Vapi API base URL (default: 'https://api.vapi.ai')",
				Default:     "https://api.vapi.ai",
			},
			{
				Name:        "assistant_id",
				Type:        "string",
				Required:    false,
				Description: "Vapi assistant ID (optional, can be created via API)",
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
