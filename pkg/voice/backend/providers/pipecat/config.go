package pipecat

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// PipecatConfig extends the base Config with Pipecat-specific configuration.
// Pipecat integrates with Daily.co for WebRTC infrastructure.
type PipecatConfig struct {
	*iface.Config
	DailyAPIKey      string        `mapstructure:"daily_api_key" yaml:"daily_api_key" validate:"required"`
	DailyAPIURL      string        `mapstructure:"daily_api_url" yaml:"daily_api_url" validate:"url" default:"https://api.daily.co/v1"`
	PipecatServerURL string        `mapstructure:"pipecat_server_url" yaml:"pipecat_server_url" validate:"omitempty,url"`
	RoomNamePrefix   string        `mapstructure:"room_name_prefix" yaml:"room_name_prefix" default:"beluga-"`
	RecordingType    string        `mapstructure:"recording_type" yaml:"recording_type" validate:"omitempty,oneof=cloud local" default:"cloud"`
	MaxParticipants  int           `mapstructure:"max_participants" yaml:"max_participants" validate:"omitempty,min=1" default:"0"`
	RoomExpiration   time.Duration `mapstructure:"room_expiration" yaml:"room_expiration" validate:"omitempty,min=1m" default:"24h"`
	EnableRecording  bool          `mapstructure:"enable_recording" yaml:"enable_recording" default:"false"`
}

// NewPipecatConfig creates a new PipecatConfig from base Config.
func NewPipecatConfig(config *iface.Config) *PipecatConfig {
	pipecatConfig := &PipecatConfig{
		Config:          config,
		DailyAPIURL:     "https://api.daily.co/v1",
		RoomNamePrefix:  "beluga-",
		EnableRecording: false,
		RecordingType:   "cloud",
		MaxParticipants: 0, // unlimited
		RoomExpiration:  24 * time.Hour,
	}

	// Extract Pipecat-specific config from ProviderConfig
	if config.ProviderConfig != nil {
		if apiKey, ok := config.ProviderConfig["daily_api_key"].(string); ok {
			pipecatConfig.DailyAPIKey = apiKey
		}
		if apiURL, ok := config.ProviderConfig["daily_api_url"].(string); ok {
			pipecatConfig.DailyAPIURL = apiURL
		}
		if serverURL, ok := config.ProviderConfig["pipecat_server_url"].(string); ok {
			pipecatConfig.PipecatServerURL = serverURL
		}
		if prefix, ok := config.ProviderConfig["room_name_prefix"].(string); ok {
			pipecatConfig.RoomNamePrefix = prefix
		}
		if enableRecording, ok := config.ProviderConfig["enable_recording"].(bool); ok {
			pipecatConfig.EnableRecording = enableRecording
		}
		if recordingType, ok := config.ProviderConfig["recording_type"].(string); ok {
			pipecatConfig.RecordingType = recordingType
		}
		if maxParticipants, ok := config.ProviderConfig["max_participants"].(int); ok {
			pipecatConfig.MaxParticipants = maxParticipants
		}
		if expiration, ok := config.ProviderConfig["room_expiration"].(time.Duration); ok {
			pipecatConfig.RoomExpiration = expiration
		}
	}

	return pipecatConfig
}
