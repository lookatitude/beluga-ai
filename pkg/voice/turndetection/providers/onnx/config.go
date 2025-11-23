package onnx

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
)

// ONNXConfig extends the base Turn Detection config with ONNX-specific settings
type ONNXConfig struct {
	*turndetection.Config

	// ModelPath specifies the path to the ONNX model file
	ModelPath string `mapstructure:"model_path" yaml:"model_path" default:"turn_detection.onnx"`

	// Threshold specifies the turn detection threshold (0.0-1.0, default: 0.5)
	Threshold float64 `mapstructure:"threshold" yaml:"threshold" default:"0.5" validate:"gte=0.0,lte=1.0"`

	// MinSilenceDuration specifies the minimum duration of silence to detect a turn end (ms)
	MinSilenceDuration time.Duration `mapstructure:"min_silence_duration" yaml:"min_silence_duration" default:"500ms"`

	// FrameSize specifies the frame size in samples
	FrameSize int `mapstructure:"frame_size" yaml:"frame_size" default:"512" validate:"min=64,max=4096"`

	// SampleRate specifies the audio sample rate in Hz
	SampleRate int `mapstructure:"sample_rate" yaml:"sample_rate" default:"16000" validate:"oneof=8000 16000 22050 24000 32000 44100 48000"`
}

// DefaultONNXConfig returns a default ONNX Turn Detection configuration
func DefaultONNXConfig() *ONNXConfig {
	return &ONNXConfig{
		Config:             turndetection.DefaultConfig(),
		ModelPath:          "turn_detection.onnx",
		Threshold:          0.5,
		MinSilenceDuration: 500 * time.Millisecond,
		FrameSize:          512,
		SampleRate:         16000,
	}
}
