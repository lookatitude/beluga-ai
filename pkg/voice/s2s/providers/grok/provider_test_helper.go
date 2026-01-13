package grok

import (
	"errors"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
)

// NewGrokVoiceProviderWithEndpoint creates a new Grok provider with a custom API endpoint.
// This is useful for testing with mock HTTP servers.
func NewGrokVoiceProviderWithEndpoint(config *s2s.Config, endpoint string) (iface.S2SProvider, error) {
	if config == nil {
		return nil, s2s.NewS2SError("NewGrokVoiceProviderWithEndpoint", s2s.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to Grok Voice config
	grokConfig := &GrokVoiceConfig{
		Config:      config,
		APIEndpoint: endpoint, // Use custom endpoint
	}

	// Set defaults if not provided
	if grokConfig.Model == "" {
		grokConfig.Model = "grok-voice-agent"
	}
	if grokConfig.VoiceID == "" {
		grokConfig.VoiceID = "alloy"
	}
	if grokConfig.LanguageCode == "" {
		grokConfig.LanguageCode = "en-US"
	}
	if grokConfig.Timeout == 0 {
		grokConfig.Timeout = 30 * time.Second
	}
	if grokConfig.SampleRate == 0 {
		grokConfig.SampleRate = 24000
	}
	if grokConfig.AudioFormat == "" {
		grokConfig.AudioFormat = "pcm"
	}
	if grokConfig.Temperature == 0 {
		grokConfig.Temperature = 0.8
	}

	// Validate API key
	if grokConfig.APIKey == "" {
		return nil, s2s.NewS2SError("NewGrokVoiceProviderWithEndpoint", s2s.ErrCodeInvalidConfig,
			errors.New("API key is required"))
	}

	provider := &GrokVoiceProvider{
		config:       grokConfig,
		providerName: "grok",
	}

	// Set default HTTP client if not provided
	if provider.httpClient == nil {
		provider.httpClient = &http.Client{Timeout: grokConfig.Timeout}
	}

	return provider, nil
}
