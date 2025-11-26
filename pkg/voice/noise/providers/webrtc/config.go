package webrtc

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
)

// WebRTCNoiseConfig extends the base Noise Cancellation config with WebRTC-specific settings.
type WebRTCNoiseConfig struct {
	*noise.Config

	// Aggressiveness specifies the noise suppression aggressiveness (0-3, default: 2)
	Aggressiveness int `mapstructure:"aggressiveness" yaml:"aggressiveness" default:"2" validate:"min=0,max=3"`

	// EnableHighPassFilter enables high-pass filtering
	EnableHighPassFilter bool `mapstructure:"enable_high_pass_filter" yaml:"enable_high_pass_filter" default:"true"`

	// EnableEchoCancellation enables echo cancellation
	EnableEchoCancellation bool `mapstructure:"enable_echo_cancellation" yaml:"enable_echo_cancellation" default:"false"`

	// EnableGainControl enables automatic gain control
	EnableGainControl bool `mapstructure:"enable_gain_control" yaml:"enable_gain_control" default:"false"`
}

// DefaultWebRTCNoiseConfig returns a default WebRTC noise cancellation configuration.
func DefaultWebRTCNoiseConfig() *WebRTCNoiseConfig {
	return &WebRTCNoiseConfig{
		Config:                 noise.DefaultConfig(),
		Aggressiveness:         2,
		EnableHighPassFilter:   true,
		EnableEchoCancellation: false,
		EnableGainControl:      false,
	}
}
