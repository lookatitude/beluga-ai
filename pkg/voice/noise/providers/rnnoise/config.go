package rnnoise

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
)

// RNNoiseConfig extends the base Noise Cancellation config with RNNoise-specific settings
type RNNoiseConfig struct {
	*noise.Config

	// ModelPath specifies the path to the RNNoise model file
	ModelPath string `mapstructure:"model_path" yaml:"model_path" default:"rnnoise.rnn"`

	// FrameSize specifies the frame size in samples (RNNoise uses 480 samples)
	FrameSize int `mapstructure:"frame_size" yaml:"frame_size" default:"480" validate:"eq=480"`

	// SampleRate must be 48000 for RNNoise
	SampleRate int `mapstructure:"sample_rate" yaml:"sample_rate" default:"48000" validate:"eq=48000"`
}

// DefaultRNNoiseConfig returns a default RNNoise configuration
func DefaultRNNoiseConfig() *RNNoiseConfig {
	return &RNNoiseConfig{
		Config:     noise.DefaultConfig(),
		ModelPath:  "rnnoise.rnn",
		FrameSize:  480,
		SampleRate: 48000,
	}
}
