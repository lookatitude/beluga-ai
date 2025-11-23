package webrtc

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

// WebRTCConfig extends the base VAD config with WebRTC VAD specific settings
type WebRTCConfig struct {
	*vad.Config

	// Mode specifies the VAD mode (0=quality, 1=low-bitrate, 2=aggressive, 3=very-aggressive)
	Mode int `mapstructure:"mode" yaml:"mode" default:"0" validate:"gte=0,lte=3"`

	// SampleRate specifies the audio sample rate (must be 8000, 16000, 32000, or 48000 for WebRTC)
	SampleRate int `mapstructure:"sample_rate" yaml:"sample_rate" default:"16000" validate:"oneof=8000 16000 32000 48000"`

	// FrameSize specifies the frame size in samples (must be 10ms, 20ms, or 30ms)
	FrameSize int `mapstructure:"frame_size" yaml:"frame_size" default:"320" validate:"min=80,max=1440"`

	// MinSpeechDuration specifies the minimum duration of speech to trigger detection (ms)
	MinSpeechDuration time.Duration `mapstructure:"min_speech_duration" yaml:"min_speech_duration" default:"250ms"`

	// MaxSilenceDuration specifies the maximum duration of silence before ending speech (ms)
	MaxSilenceDuration time.Duration `mapstructure:"max_silence_duration" yaml:"max_silence_duration" default:"500ms"`
}

// DefaultWebRTCConfig returns a default WebRTC VAD configuration
func DefaultWebRTCConfig() *WebRTCConfig {
	return &WebRTCConfig{
		Config:             vad.DefaultConfig(),
		Mode:               0,
		SampleRate:         16000,
		FrameSize:          320, // 20ms at 16kHz
		MinSpeechDuration:  250 * time.Millisecond,
		MaxSilenceDuration: 500 * time.Millisecond,
	}
}
