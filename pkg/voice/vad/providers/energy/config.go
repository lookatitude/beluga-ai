package energy

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

// EnergyConfig extends the base VAD config with Energy-based VAD specific settings
type EnergyConfig struct {
	*vad.Config

	// Threshold specifies the energy threshold for speech detection
	Threshold float64 `mapstructure:"threshold" yaml:"threshold" default:"0.01" validate:"gte=0.0"`

	// EnergyWindowSize specifies the window size for energy calculation (samples)
	EnergyWindowSize int `mapstructure:"energy_window_size" yaml:"energy_window_size" default:"256" validate:"min=64,max=2048"`

	// MinSpeechDuration specifies the minimum duration of speech to trigger detection (ms)
	MinSpeechDuration time.Duration `mapstructure:"min_speech_duration" yaml:"min_speech_duration" default:"250ms"`

	// MaxSilenceDuration specifies the maximum duration of silence before ending speech (ms)
	MaxSilenceDuration time.Duration `mapstructure:"max_silence_duration" yaml:"max_silence_duration" default:"500ms"`

	// AdaptiveThreshold enables adaptive threshold based on background noise
	AdaptiveThreshold bool `mapstructure:"adaptive_threshold" yaml:"adaptive_threshold" default:"true"`

	// NoiseFloor specifies the noise floor for adaptive threshold (0.0-1.0)
	NoiseFloor float64 `mapstructure:"noise_floor" yaml:"noise_floor" default:"0.001" validate:"gte=0.0,lte=1.0"`
}

// DefaultEnergyConfig returns a default Energy-based VAD configuration
func DefaultEnergyConfig() *EnergyConfig {
	return &EnergyConfig{
		Config:             vad.DefaultConfig(),
		Threshold:          0.01,
		EnergyWindowSize:   256,
		MinSpeechDuration:  250 * time.Millisecond,
		MaxSilenceDuration: 500 * time.Millisecond,
		AdaptiveThreshold:  true,
		NoiseFloor:         0.001,
	}
}
