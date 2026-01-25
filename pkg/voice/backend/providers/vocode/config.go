package vocode

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// VocodeConfig extends the base Config with Vocode-specific configuration.
type VocodeConfig struct {
	*iface.Config
	APIKey          string        `mapstructure:"api_key" yaml:"api_key" validate:"required"`
	APIURL          string        `mapstructure:"api_url" yaml:"api_url" validate:"url" default:"https://api.vocode.dev"`
	AgentID         string        `mapstructure:"agent_id" yaml:"agent_id"`
	PhoneNumberID   string        `mapstructure:"phone_number_id" yaml:"phone_number_id"`
	RecordingType   string        `mapstructure:"recording_type" yaml:"recording_type" validate:"omitempty,oneof=transcript audio" default:"transcript"`
	MaxCallDuration time.Duration `mapstructure:"max_call_duration" yaml:"max_call_duration" validate:"omitempty,min=1m" default:"30m"`
	EnableRecording bool          `mapstructure:"enable_recording" yaml:"enable_recording" default:"false"`
}

// NewVocodeConfig creates a new VocodeConfig from base Config.
func NewVocodeConfig(config *iface.Config) *VocodeConfig {
	vocodeConfig := &VocodeConfig{
		Config:          config,
		APIURL:          "https://api.vocode.dev",
		EnableRecording: false,
		RecordingType:   "transcript",
		MaxCallDuration: 30 * time.Minute,
	}

	// Extract Vocode-specific config from ProviderConfig
	if config.ProviderConfig != nil {
		if apiKey, ok := config.ProviderConfig["api_key"].(string); ok {
			vocodeConfig.APIKey = apiKey
		}
		if apiURL, ok := config.ProviderConfig["api_url"].(string); ok {
			vocodeConfig.APIURL = apiURL
		}
		if agentID, ok := config.ProviderConfig["agent_id"].(string); ok {
			vocodeConfig.AgentID = agentID
		}
		if phoneNumberID, ok := config.ProviderConfig["phone_number_id"].(string); ok {
			vocodeConfig.PhoneNumberID = phoneNumberID
		}
		if enableRecording, ok := config.ProviderConfig["enable_recording"].(bool); ok {
			vocodeConfig.EnableRecording = enableRecording
		}
		if recordingType, ok := config.ProviderConfig["recording_type"].(string); ok {
			vocodeConfig.RecordingType = recordingType
		}
		if maxCallDuration, ok := config.ProviderConfig["max_call_duration"].(time.Duration); ok {
			vocodeConfig.MaxCallDuration = maxCallDuration
		}
	}

	return vocodeConfig
}
