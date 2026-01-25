package vapi

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// VapiConfig extends the base Config with Vapi-specific configuration.
type VapiConfig struct {
	*iface.Config
	APIKey          string        `mapstructure:"api_key" yaml:"api_key" validate:"required"`
	APIURL          string        `mapstructure:"api_url" yaml:"api_url" validate:"url" default:"https://api.vapi.ai"`
	AssistantID     string        `mapstructure:"assistant_id" yaml:"assistant_id"`
	PhoneNumberID   string        `mapstructure:"phone_number_id" yaml:"phone_number_id"`
	RecordingType   string        `mapstructure:"recording_type" yaml:"recording_type" validate:"omitempty,oneof=transcript audio" default:"transcript"`
	MaxCallDuration time.Duration `mapstructure:"max_call_duration" yaml:"max_call_duration" validate:"omitempty,min=1m" default:"30m"`
	EnableRecording bool          `mapstructure:"enable_recording" yaml:"enable_recording" default:"false"`
}

// NewVapiConfig creates a new VapiConfig from base Config.
func NewVapiConfig(config *iface.Config) *VapiConfig {
	vapiConfig := &VapiConfig{
		Config:          config,
		APIURL:          "https://api.vapi.ai",
		EnableRecording: false,
		RecordingType:   "transcript",
		MaxCallDuration: 30 * time.Minute,
	}

	// Extract Vapi-specific config from ProviderConfig
	if config.ProviderConfig != nil {
		if apiKey, ok := config.ProviderConfig["api_key"].(string); ok {
			vapiConfig.APIKey = apiKey
		}
		if apiURL, ok := config.ProviderConfig["api_url"].(string); ok {
			vapiConfig.APIURL = apiURL
		}
		if assistantID, ok := config.ProviderConfig["assistant_id"].(string); ok {
			vapiConfig.AssistantID = assistantID
		}
		if phoneNumberID, ok := config.ProviderConfig["phone_number_id"].(string); ok {
			vapiConfig.PhoneNumberID = phoneNumberID
		}
		if enableRecording, ok := config.ProviderConfig["enable_recording"].(bool); ok {
			vapiConfig.EnableRecording = enableRecording
		}
		if recordingType, ok := config.ProviderConfig["recording_type"].(string); ok {
			vapiConfig.RecordingType = recordingType
		}
		if maxCallDuration, ok := config.ProviderConfig["max_call_duration"].(time.Duration); ok {
			vapiConfig.MaxCallDuration = maxCallDuration
		}
	}

	return vapiConfig
}
