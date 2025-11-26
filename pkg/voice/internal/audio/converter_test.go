package audio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConverter(t *testing.T) {
	converter := NewConverter()
	assert.NotNil(t, converter)
}

func TestConverter_Convert(t *testing.T) {
	tests := []struct {
		from    *AudioFormat
		to      *AudioFormat
		name    string
		errMsg  string
		data    []byte
		wantErr bool
	}{
		{
			name: "same format - no conversion needed",
			data: []byte{1, 2, 3, 4},
			from: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			to: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			wantErr: false,
		},
		{
			name: "nil source format",
			data: []byte{1, 2, 3, 4},
			from: nil,
			to: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "source and target formats must be non-nil",
		},
		{
			name: "nil target format",
			data: []byte{1, 2, 3, 4},
			from: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			to:      nil,
			wantErr: true,
			errMsg:  "source and target formats must be non-nil",
		},
		{
			name: "invalid source format",
			data: []byte{1, 2, 3, 4},
			from: &AudioFormat{
				SampleRate: 0, // Invalid
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			to: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "invalid source format",
		},
		{
			name: "invalid target format",
			data: []byte{1, 2, 3, 4},
			from: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			to: &AudioFormat{
				SampleRate: 0, // Invalid
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "invalid target format",
		},
		{
			name: "different sample rate - conversion not implemented",
			data: []byte{1, 2, 3, 4},
			from: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			to: &AudioFormat{
				SampleRate: 48000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "audio format conversion not yet implemented",
		},
		{
			name: "different channels - conversion not implemented",
			data: []byte{1, 2, 3, 4},
			from: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			to: &AudioFormat{
				SampleRate: 16000,
				Channels:   2,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "audio format conversion not yet implemented",
		},
		{
			name: "different encoding - conversion not implemented",
			data: []byte{1, 2, 3, 4},
			from: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			to: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "opus",
			},
			wantErr: true,
			errMsg:  "audio format conversion not yet implemented",
		},
		{
			name: "different bit depth - conversion not implemented",
			data: []byte{1, 2, 3, 4},
			from: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "pcm",
			},
			to: &AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   24,
				Encoding:   "pcm",
			},
			wantErr: true,
			errMsg:  "audio format conversion not yet implemented",
		},
	}

	converter := NewConverter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.data, tt.from, tt.to)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.data, result)
			}
		})
	}
}
