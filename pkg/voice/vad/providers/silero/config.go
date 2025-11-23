package silero

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

// SileroConfig extends the base VAD config with Silero VAD specific settings
type SileroConfig struct {
	*vad.Config

	// ModelPath specifies the path to the Silero VAD ONNX model file
	ModelPath string `mapstructure:"model_path" yaml:"model_path" default:"silero_vad.onnx"`

	// Threshold specifies the speech detection threshold (0.0-1.0, default: 0.5)
	Threshold float64 `mapstructure:"threshold" yaml:"threshold" default:"0.5" validate:"gte=0.0,lte=1.0"`

	// MinSpeechDuration specifies the minimum duration of speech to trigger detection (ms)
	MinSpeechDuration time.Duration `mapstructure:"min_speech_duration" yaml:"min_speech_duration" default:"250ms"`

	// MaxSilenceDuration specifies the maximum duration of silence before ending speech (ms)
	MaxSilenceDuration time.Duration `mapstructure:"max_silence_duration" yaml:"max_silence_duration" default:"500ms"`

	// SampleRate specifies the audio sample rate (must be 8000 or 16000 for Silero)
	SampleRate int `mapstructure:"sample_rate" yaml:"sample_rate" default:"16000" validate:"oneof=8000 16000"`

	// FrameSize specifies the frame size in samples (default: 512 for 16kHz, 256 for 8kHz)
	FrameSize int `mapstructure:"frame_size" yaml:"frame_size" default:"512" validate:"min=64,max=4096"`
}

// DefaultSileroConfig returns a default Silero VAD configuration
func DefaultSileroConfig() *SileroConfig {
	return &SileroConfig{
		Config:             vad.DefaultConfig(),
		ModelPath:          "silero_vad.onnx",
		Threshold:          0.5,
		MinSpeechDuration:  250 * time.Millisecond,
		MaxSilenceDuration: 500 * time.Millisecond,
		SampleRate:         16000,
		FrameSize:          512,
	}
}
