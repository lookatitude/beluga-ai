// Package audio provides audio format utilities for the voice package.
package audio

import (
	"fmt"
)

// AudioFormat represents audio format specifications.
type AudioFormat struct {
	SampleRate int    // Sample rate in Hz (e.g., 16000, 48000)
	Channels   int    // Number of channels (1 for mono, 2 for stereo)
	BitDepth   int    // Bit depth (16, 24, 32)
	Encoding   string // Encoding format ("pcm", "opus", "mp3", etc.)
}

// Validate validates the audio format.
func (af *AudioFormat) Validate() error {
	if af.SampleRate <= 0 {
		return fmt.Errorf("sample rate must be greater than 0")
	}

	if af.Channels != 1 && af.Channels != 2 {
		return fmt.Errorf("channels must be 1 (mono) or 2 (stereo), got: %d", af.Channels)
	}

	if af.BitDepth != 16 && af.BitDepth != 24 && af.BitDepth != 32 {
		return fmt.Errorf("bit depth must be 16, 24, or 32, got: %d", af.BitDepth)
	}

	if af.Encoding == "" {
		return fmt.Errorf("encoding must be non-empty")
	}

	return nil
}

// DefaultAudioFormat returns a default audio format (16kHz, mono, 16-bit PCM).
func DefaultAudioFormat() *AudioFormat {
	return &AudioFormat{
		SampleRate: 16000,
		Channels:   1,
		BitDepth:   16,
		Encoding:   "pcm",
	}
}
