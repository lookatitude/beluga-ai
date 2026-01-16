package twilio

import (
	"context"
	"errors"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// TwilioProvider implements the BackendProvider interface for Twilio.
type TwilioProvider struct{}

// NewTwilioProvider creates a new Twilio provider.
func NewTwilioProvider() *TwilioProvider {
	return &TwilioProvider{}
}

// GetName returns the provider name.
func (p *TwilioProvider) GetName() string {
	return "twilio"
}

// GetCapabilities returns the capabilities of the Twilio provider.
func (p *TwilioProvider) GetCapabilities(ctx context.Context) (*vbiface.ProviderCapabilities, error) {
	return &vbiface.ProviderCapabilities{
		S2SSupport:            false, // Twilio uses STT/TTS pipeline
		MultiUserSupport:      true,
		SessionPersistence:    true,
		CustomAuth:            false,
		CustomRateLimiting:    false,           // Twilio handles rate limiting
		MaxConcurrentSessions: 100,             // SC-003: Support 100 concurrent calls
		MinLatency:            2 * time.Second, // FR-009: <2s latency target
		SupportedCodecs:       []string{"mu-law", "pcmu"},
	}, nil
}

// CreateBackend creates a new Twilio backend instance.
func (p *TwilioProvider) CreateBackend(ctx context.Context, config *vbiface.Config) (vbiface.VoiceBackend, error) {
	twilioConfig := NewTwilioConfig(config)

	// Validate Twilio-specific config
	if err := p.ValidateConfig(ctx, config); err != nil {
		return nil, err
	}

	return NewTwilioBackend(twilioConfig)
}

// ValidateConfig validates the Twilio provider configuration.
func (p *TwilioProvider) ValidateConfig(ctx context.Context, config *vbiface.Config) error {
	// Validate base config
	if err := backend.ValidateConfig(config); err != nil {
		return err
	}

	// Validate Twilio-specific fields
	twilioConfig := NewTwilioConfig(config)

	if err := twilioConfig.Validate(); err != nil {
		return err
	}

	// Validate pipeline type (Twilio uses STT_TTS)
	if config.PipelineType != vbiface.PipelineTypeSTTTTS {
		return errors.New("twilio provider requires STT_TTS pipeline type")
	}

	// Validate STT and TTS providers are set
	if config.STTProvider == "" {
		return errors.New("stt_provider is required for Twilio provider")
	}

	if config.TTSProvider == "" {
		return errors.New("tts_provider is required for Twilio provider")
	}

	return nil
}

// GetConfigSchema returns the configuration schema for the Twilio provider.
func (p *TwilioProvider) GetConfigSchema() *vbiface.ConfigSchema {
	return &vbiface.ConfigSchema{
		Fields: []vbiface.ConfigField{
			{
				Name:        "provider",
				Type:        "string",
				Required:    true,
				Description: "Provider name (must be 'twilio')",
			},
			{
				Name:        "account_sid",
				Type:        "string",
				Required:    true,
				Description: "Twilio Account SID",
			},
			{
				Name:        "auth_token",
				Type:        "string",
				Required:    true,
				Description: "Twilio Auth Token",
			},
			{
				Name:        "phone_number",
				Type:        "string",
				Required:    true,
				Description: "Twilio phone number (E.164 format)",
			},
			{
				Name:        "webhook_url",
				Type:        "string",
				Required:    false,
				Description: "Webhook URL for call events",
			},
			{
				Name:        "status_callback_url",
				Type:        "string",
				Required:    false,
				Description: "Status callback URL for call status updates",
			},
			{
				Name:        "account_name",
				Type:        "string",
				Required:    false,
				Description: "Account name for multi-account support (FR-041)",
			},
		},
	}
}
