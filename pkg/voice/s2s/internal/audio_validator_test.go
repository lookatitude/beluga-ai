package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAudioFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  AudioFormat
		wantErr bool
	}{
		{
			name: "valid format",
			format: AudioFormat{
				SampleRate: 24000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "PCM",
			},
			wantErr: false,
		},
		{
			name: "invalid sample rate",
			format: AudioFormat{
				SampleRate: 44100,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "PCM",
			},
			wantErr: true,
		},
		{
			name: "invalid channels",
			format: AudioFormat{
				SampleRate: 24000,
				Channels:   3,
				BitDepth:   16,
				Encoding:   "PCM",
			},
			wantErr: true,
		},
		{
			name: "invalid bit depth",
			format: AudioFormat{
				SampleRate: 24000,
				Channels:   1,
				BitDepth:   64,
				Encoding:   "PCM",
			},
			wantErr: true,
		},
		{
			name: "empty encoding",
			format: AudioFormat{
				SampleRate: 24000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAudioFormat(tt.format)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateAudioQuality(t *testing.T) {
	tests := []struct {
		name    string
		quality AudioQuality
		wantErr bool
	}{
		{
			name: "valid quality",
			quality: AudioQuality{
				SNR:        30.0,
				IsClear:    true,
				NoiseLevel: 0.1,
			},
			wantErr: false,
		},
		{
			name: "invalid SNR (too low)",
			quality: AudioQuality{
				SNR:        -100.0,
				IsClear:    true,
				NoiseLevel: 0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid SNR (too high)",
			quality: AudioQuality{
				SNR:        200.0,
				IsClear:    true,
				NoiseLevel: 0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid noise level (negative)",
			quality: AudioQuality{
				SNR:        30.0,
				IsClear:    true,
				NoiseLevel: -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid noise level (too high)",
			quality: AudioQuality{
				SNR:        30.0,
				IsClear:    true,
				NoiseLevel: 1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAudioQuality(tt.quality)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateAudioInput(t *testing.T) {
	tests := []struct {
		name    string
		input   *AudioInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: &AudioInput{
				Data: []byte{1, 2, 3, 4, 5},
				Format: AudioFormat{
					SampleRate: 24000,
					Channels:   1,
					BitDepth:   16,
					Encoding:   "PCM",
				},
				Quality: AudioQuality{
					SNR:        30.0,
					IsClear:    true,
					NoiseLevel: 0.1,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
		{
			name: "empty audio data",
			input: &AudioInput{
				Data: []byte{},
				Format: AudioFormat{
					SampleRate: 24000,
					Channels:   1,
					BitDepth:   16,
					Encoding:   "PCM",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			input: &AudioInput{
				Data: []byte{1, 2, 3, 4, 5},
				Format: AudioFormat{
					SampleRate: 44100, // Invalid
					Channels:   1,
					BitDepth:   16,
					Encoding:   "PCM",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAudioInput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateAudioOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  *AudioOutput
		wantErr bool
	}{
		{
			name: "valid output",
			output: &AudioOutput{
				Data: []byte{1, 2, 3, 4, 5},
				Format: AudioFormat{
					SampleRate: 24000,
					Channels:   1,
					BitDepth:   16,
					Encoding:   "PCM",
				},
			},
			wantErr: false,
		},
		{
			name:    "nil output",
			output:  nil,
			wantErr: true,
		},
		{
			name: "empty audio data",
			output: &AudioOutput{
				Data: []byte{},
				Format: AudioFormat{
					SampleRate: 24000,
					Channels:   1,
					BitDepth:   16,
					Encoding:   "PCM",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			output: &AudioOutput{
				Data: []byte{1, 2, 3, 4, 5},
				Format: AudioFormat{
					SampleRate: 44100, // Invalid
					Channels:   1,
					BitDepth:   16,
					Encoding:   "PCM",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAudioOutput(tt.output)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
