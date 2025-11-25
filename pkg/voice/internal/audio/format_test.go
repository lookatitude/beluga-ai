package audio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAudioFormat(t *testing.T) {
	format := DefaultAudioFormat()

	assert.NotNil(t, format)
	assert.Equal(t, 16000, format.SampleRate)
	assert.Equal(t, 1, format.Channels)
	assert.Equal(t, 16, format.BitDepth)
	assert.Equal(t, "pcm", format.Encoding)
}

func TestAudioFormat_Validate(t *testing.T) {
	tests := []struct {
		name    string
		format  *AudioFormat
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid format",
			format: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			wantErr: false,
		},
		{
			name: "valid stereo format",
			format: &AudioFormat{
				SampleRate: 48000,
				Channels:   2,
				BitDepth:   24,
				Encoding:   "opus",
			},
			wantErr: false,
		},
		{
			name: "invalid sample rate - zero",
			format: &AudioFormat{
				SampleRate: 0,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "sample rate must be greater than 0",
		},
		{
			name: "invalid sample rate - negative",
			format: &AudioFormat{
				SampleRate: -16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "sample rate must be greater than 0",
		},
		{
			name: "invalid channels - zero",
			format: &AudioFormat{
				SampleRate: 16000,
				Channels:   0,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "channels must be 1 (mono) or 2 (stereo)",
		},
		{
			name: "invalid channels - too many",
			format: &AudioFormat{
				SampleRate: 16000,
				Channels:   5,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "channels must be 1 (mono) or 2 (stereo)",
		},
		{
			name: "invalid bit depth - 8",
			format: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   8,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "bit depth must be 16, 24, or 32",
		},
		{
			name: "invalid bit depth - 32 but wrong value",
			format: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   64,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "bit depth must be 16, 24, or 32",
		},
		{
			name: "valid 24-bit depth",
			format: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   24,
				Encoding:   "pcm",
			},
			wantErr: false,
		},
		{
			name: "valid 32-bit depth",
			format: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   32,
				Encoding:   "pcm",
			},
			wantErr: false,
		},
		{
			name: "invalid encoding - empty",
			format: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "",
			},
			wantErr: true,
			errMsg:  "encoding must be non-empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.format.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
