package rnnoise

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

// RNNoiseConfig extends the base VAD config with RNNoise VAD specific settings
type RNNoiseConfig struct {
	*vad.Config

	// ModelPath specifies the path to the RNNoise model file
	ModelPath string `mapstructure:"model_path" yaml:"model_path" default:"rnnoise_model.rnn"`

	// Threshold specifies the speech detection threshold (0.0-1.0, default: 0.5)
	Threshold float64 `mapstructure:"threshold" yaml:"threshold" default:"0.5" validate:"gte=0.0,lte=1.0"`

	// SampleRate specifies the audio sample rate (must be 48000 for RNNoise)
	SampleRate int `mapstructure:"sample_rate" yaml:"sample_rate" default:"48000" validate:"eq=48000"`

	// FrameSize specifies the frame size in samples (must be 480 for RNNoise, 10ms at 48kHz)
	FrameSize int `mapstructure:"frame_size" yaml:"frame_size" default:"480" validate:"eq=480"`

	// MinSpeechDuration specifies the minimum duration of speech to trigger detection (ms)
	MinSpeechDuration time.Duration `mapstructure:"min_speech_duration" yaml:"min_speech_duration" default:"250ms"`

	// MaxSilenceDuration specifies the maximum duration of silence before ending speech (ms)
	MaxSilenceDuration time.Duration `mapstructure:"max_silence_duration" yaml:"max_silence_duration" default:"500ms"`
}

// DefaultRNNoiseConfig returns a default RNNoise VAD configuration
func DefaultRNNoiseConfig() *RNNoiseConfig {
	return &RNNoiseConfig{
		Config:             vad.DefaultConfig(),
		ModelPath:          "rnnoise_model.rnn",
		Threshold:          0.5,
		SampleRate:         48000,
		FrameSize:          480,
		MinSpeechDuration:  250 * time.Millisecond,
		MaxSilenceDuration: 500 * time.Millisecond,
	}
}
