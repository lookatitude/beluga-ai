package gemini

import (
	"errors"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
)

// NewGeminiNativeProviderWithEndpoint creates a new Gemini provider with a custom API endpoint.
// This is useful for testing with mock HTTP servers.
func NewGeminiNativeProviderWithEndpoint(config *s2s.Config, endpoint string) (iface.S2SProvider, error) {
	if config == nil {
		return nil, s2s.NewS2SError("NewGeminiNativeProviderWithEndpoint", s2s.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to Gemini Native config
	geminiConfig := &GeminiNativeConfig{
		Config:      config,
		APIEndpoint: endpoint, // Use custom endpoint
	}

	// Set defaults if not provided
	if geminiConfig.Model == "" {
		geminiConfig.Model = "gemini-2.5-flash"
	}
	if geminiConfig.Location == "" {
		geminiConfig.Location = "us-central1"
	}
	if geminiConfig.VoiceID == "" {
		geminiConfig.VoiceID = "default"
	}
	if geminiConfig.LanguageCode == "" {
		geminiConfig.LanguageCode = "en-US"
	}
	if geminiConfig.Timeout == 0 {
		geminiConfig.Timeout = 30 * time.Second
	}
	if geminiConfig.SampleRate == 0 {
		geminiConfig.SampleRate = 24000
	}
	if geminiConfig.AudioFormat == "" {
		geminiConfig.AudioFormat = "pcm"
	}
	if geminiConfig.Temperature == 0 {
		geminiConfig.Temperature = 0.8
	}

	// Validate API key
	if geminiConfig.APIKey == "" {
		return nil, s2s.NewS2SError("NewGeminiNativeProviderWithEndpoint", s2s.ErrCodeInvalidConfig,
			errors.New("API key is required"))
	}

	provider := &GeminiNativeProvider{
		config:       geminiConfig,
		providerName: "gemini",
	}

	// Set default HTTP client
	provider.httpClient = &http.Client{Timeout: geminiConfig.Timeout}

	return provider, nil
}
