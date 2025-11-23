package audio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCodec(t *testing.T) {
	codec := NewCodec()
	assert.NotNil(t, codec)
}

func TestSupportedCodecs(t *testing.T) {
	codec := NewCodec()
	supported := codec.SupportedCodecs()

	assert.NotEmpty(t, supported)
	assert.Contains(t, supported, "pcm")
	assert.Contains(t, supported, "opus")
	assert.Contains(t, supported, "mp3")
	assert.Contains(t, supported, "wav")
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		name     string
		codec    string
		expected bool
	}{
		{
			name:     "pcm is supported",
			codec:    "pcm",
			expected: true,
		},
		{
			name:     "opus is supported",
			codec:    "opus",
			expected: true,
		},
		{
			name:     "mp3 is supported",
			codec:    "mp3",
			expected: true,
		},
		{
			name:     "wav is supported",
			codec:    "wav",
			expected: true,
		},
		{
			name:     "aac is not supported",
			codec:    "aac",
			expected: false,
		},
		{
			name:     "empty string is not supported",
			codec:    "",
			expected: false,
		},
		{
			name:     "case sensitive - PCM is not supported",
			codec:    "PCM",
			expected: false,
		},
	}

	codec := NewCodec()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := codec.IsSupported(tt.codec)
			assert.Equal(t, tt.expected, result)
		})
	}
}

