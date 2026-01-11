package livekit

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// LiveKitConfig extends the base Config with LiveKit-specific fields.
type LiveKitConfig struct {
	*iface.Config

	// LiveKit-specific configuration
	APIKey    string `mapstructure:"api_key" yaml:"api_key" validate:"required"`
	APISecret string `mapstructure:"api_secret" yaml:"api_secret" validate:"required"`
	URL       string `mapstructure:"url" yaml:"url" validate:"required,url"`
	RoomName  string `mapstructure:"room_name" yaml:"room_name"` // Optional, can be generated
}

// NewLiveKitConfig creates a new LiveKit configuration from base config.
func NewLiveKitConfig(baseConfig *iface.Config) *LiveKitConfig {
	config := &LiveKitConfig{
		Config: baseConfig,
	}

	// Extract LiveKit-specific config from ProviderConfig if present
	if baseConfig.ProviderConfig != nil {
		if apiKey, ok := baseConfig.ProviderConfig["api_key"].(string); ok {
			config.APIKey = apiKey
		}
		if apiSecret, ok := baseConfig.ProviderConfig["api_secret"].(string); ok {
			config.APISecret = apiSecret
		}
		if url, ok := baseConfig.ProviderConfig["url"].(string); ok {
			config.URL = url
		}
		if roomName, ok := baseConfig.ProviderConfig["room_name"].(string); ok {
			config.RoomName = roomName
		}
	}

	return config
}
