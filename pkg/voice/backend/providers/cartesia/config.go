package cartesia

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// CartesiaConfig extends the base Config with Cartesia-specific configuration.
type CartesiaConfig struct {
	*iface.Config

	// APIKey is the Cartesia API key.
	APIKey string `mapstructure:"api_key" yaml:"api_key" validate:"required"`

	// APIURL is the Cartesia API base URL (default: https://api.cartesia.ai).
	APIURL string `mapstructure:"api_url" yaml:"api_url" validate:"url" default:"https://api.cartesia.ai"`

	// ModelID is the Cartesia model ID for voice generation (optional).
	ModelID string `mapstructure:"model_id" yaml:"model_id"`

	// VoiceID is the Cartesia voice ID (optional).
	VoiceID string `mapstructure:"voice_id" yaml:"voice_id"`

	// EnableStreaming enables streaming audio generation (optional).
	EnableStreaming bool `mapstructure:"enable_streaming" yaml:"enable_streaming" default:"true"`

	// SampleRate is the audio sample rate (optional, default: 24000).
	SampleRate int `mapstructure:"sample_rate" yaml:"sample_rate" validate:"omitempty,oneof=16000 24000 48000" default:"24000"`

	// LatencyOptimization enables latency optimization (optional).
	LatencyOptimization bool `mapstructure:"latency_optimization" yaml:"latency_optimization" default:"true"`
}

// NewCartesiaConfig creates a new CartesiaConfig from base Config.
func NewCartesiaConfig(config *iface.Config) *CartesiaConfig {
	cartesiaConfig := &CartesiaConfig{
		Config:              config,
		APIURL:              "https://api.cartesia.ai",
		EnableStreaming:     true,
		SampleRate:          24000,
		LatencyOptimization: true,
	}

	// Extract Cartesia-specific config from ProviderConfig
	if config.ProviderConfig != nil {
		if apiKey, ok := config.ProviderConfig["api_key"].(string); ok {
			cartesiaConfig.APIKey = apiKey
		}
		if apiURL, ok := config.ProviderConfig["api_url"].(string); ok {
			cartesiaConfig.APIURL = apiURL
		}
		if modelID, ok := config.ProviderConfig["model_id"].(string); ok {
			cartesiaConfig.ModelID = modelID
		}
		if voiceID, ok := config.ProviderConfig["voice_id"].(string); ok {
			cartesiaConfig.VoiceID = voiceID
		}
		if enableStreaming, ok := config.ProviderConfig["enable_streaming"].(bool); ok {
			cartesiaConfig.EnableStreaming = enableStreaming
		}
		if sampleRate, ok := config.ProviderConfig["sample_rate"].(int); ok {
			cartesiaConfig.SampleRate = sampleRate
		}
		if latencyOptimization, ok := config.ProviderConfig["latency_optimization"].(bool); ok {
			cartesiaConfig.LatencyOptimization = latencyOptimization
		}
	}

	return cartesiaConfig
}
