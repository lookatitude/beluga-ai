package internal

import (
	"errors"
	"fmt"
)

// ValidateAudioFormat validates audio format parameters.
func ValidateAudioFormat(format AudioFormat) error {
	// Validate sample rate
	validSampleRates := []int{8000, 16000, 24000, 48000}
	validSampleRate := false
	for _, rate := range validSampleRates {
		if format.SampleRate == rate {
			validSampleRate = true
			break
		}
	}
	if !validSampleRate {
		return fmt.Errorf("invalid sample rate: %d (supported: %v)", format.SampleRate, validSampleRates)
	}

	// Validate channels
	if format.Channels < 1 || format.Channels > 2 {
		return fmt.Errorf("invalid channels: %d (supported: 1 or 2)", format.Channels)
	}

	// Validate bit depth
	validBitDepths := []int{8, 16, 24, 32}
	validBitDepth := false
	for _, depth := range validBitDepths {
		if format.BitDepth == depth {
			validBitDepth = true
			break
		}
	}
	if !validBitDepth {
		return fmt.Errorf("invalid bit depth: %d (supported: %v)", format.BitDepth, validBitDepths)
	}

	// Validate encoding
	if format.Encoding == "" {
		return errors.New("encoding cannot be empty")
	}

	return nil
}

// ValidateAudioQuality validates audio quality parameters.
func ValidateAudioQuality(quality AudioQuality) error {
	// Validate SNR (typically -20 to 60 dB)
	if quality.SNR < -50 || quality.SNR > 100 {
		return fmt.Errorf("invalid SNR: %.2f (expected range: -50 to 100 dB)", quality.SNR)
	}

	// Validate noise level (0.0 to 1.0)
	if quality.NoiseLevel < 0.0 || quality.NoiseLevel > 1.0 {
		return fmt.Errorf("invalid noise level: %.2f (expected range: 0.0 to 1.0)", quality.NoiseLevel)
	}

	return nil
}

// ValidateAudioInput validates audio input data and format.
func ValidateAudioInput(input *AudioInput) error {
	if input == nil {
		return errors.New("audio input cannot be nil")
	}

	if len(input.Data) == 0 {
		return errors.New("audio data cannot be empty")
	}

	// Validate format
	if err := ValidateAudioFormat(input.Format); err != nil {
		return fmt.Errorf("invalid audio format: %w", err)
	}

	// Validate quality if provided
	if err := ValidateAudioQuality(input.Quality); err != nil {
		return fmt.Errorf("invalid audio quality: %w", err)
	}

	return nil
}

// ValidateAudioOutput validates audio output data and format.
func ValidateAudioOutput(output *AudioOutput) error {
	if output == nil {
		return errors.New("audio output cannot be nil")
	}

	if len(output.Data) == 0 {
		return errors.New("audio data cannot be empty")
	}

	// Validate format
	if err := ValidateAudioFormat(output.Format); err != nil {
		return fmt.Errorf("invalid audio format: %w", err)
	}

	return nil
}
