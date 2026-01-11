package livekit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// LiveKitProvider implements the BackendProvider interface for LiveKit.
type LiveKitProvider struct{}

// NewLiveKitProvider creates a new LiveKit provider.
func NewLiveKitProvider() *LiveKitProvider {
	return &LiveKitProvider{}
}

// GetName returns the provider name.
func (p *LiveKitProvider) GetName() string {
	return "livekit"
}

// GetCapabilities returns the capabilities of the LiveKit provider.
func (p *LiveKitProvider) GetCapabilities(ctx context.Context) (*iface.ProviderCapabilities, error) {
	return &iface.ProviderCapabilities{
		S2SSupport:            true,
		MultiUserSupport:      true,
		SessionPersistence:   true,
		CustomAuth:           true,
		CustomRateLimiting:   true,
		MaxConcurrentSessions: 0, // LiveKit supports unlimited sessions
		MinLatency:           50 * time.Millisecond, // LiveKit can achieve <100ms latency
		SupportedCodecs:      []string{"opus", "pcm", "g722"},
	}, nil
}

// CreateBackend creates a new LiveKit backend instance.
func (p *LiveKitProvider) CreateBackend(ctx context.Context, config *iface.Config) (iface.VoiceBackend, error) {
	livekitConfig := NewLiveKitConfig(config)

	// Validate LiveKit-specific config
	if err := p.ValidateConfig(ctx, config); err != nil {
		return nil, err
	}

	return NewLiveKitBackend(livekitConfig)
}

// ValidateConfig validates the LiveKit provider configuration.
func (p *LiveKitProvider) ValidateConfig(ctx context.Context, config *iface.Config) error {
	// Validate base config
	if err := backend.ValidateConfig(config); err != nil {
		return err
	}

	// Validate LiveKit-specific fields
	livekitConfig := NewLiveKitConfig(config)

	if livekitConfig.APIKey == "" {
		return errors.New("api_key is required for LiveKit provider")
	}

	if livekitConfig.APISecret == "" {
		return errors.New("api_secret is required for LiveKit provider")
	}

	if livekitConfig.URL == "" {
		return errors.New("url is required for LiveKit provider")
	}

	// Validate URL format (basic check)
	if !isValidURL(livekitConfig.URL) {
		return fmt.Errorf("invalid URL format: %s", livekitConfig.URL)
	}

	return nil
}

// GetConfigSchema returns the configuration schema for the LiveKit provider.
func (p *LiveKitProvider) GetConfigSchema() *iface.ConfigSchema {
	return &iface.ConfigSchema{
		Fields: []iface.ConfigField{
			{
				Name:        "provider",
				Type:        "string",
				Required:    true,
				Description: "Provider name (must be 'livekit')",
			},
			{
				Name:        "api_key",
				Type:        "string",
				Required:    true,
				Description: "LiveKit API key",
			},
			{
				Name:        "api_secret",
				Type:        "string",
				Required:    true,
				Description: "LiveKit API secret",
			},
			{
				Name:        "url",
				Type:        "string",
				Required:    true,
				Description: "LiveKit server URL (e.g., 'wss://your-livekit-server.com')",
			},
			{
				Name:        "room_name",
				Type:        "string",
				Required:    false,
				Description: "LiveKit room name (optional, can be auto-generated)",
			},
		},
	}
}

// isValidURL performs basic URL validation.
func isValidURL(url string) bool {
	return len(url) > 0 && (url[:4] == "ws://" || url[:5] == "wss://" || url[:7] == "http://" || url[:8] == "https://")
}
