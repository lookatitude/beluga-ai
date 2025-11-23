package tts

import (
	"context"
	"io"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTTSProvider_GenerateSpeech(t *testing.T) {
	tests := []struct {
		name          string
		provider      iface.TTSProvider
		text          string
		expectedError bool
	}{
		{
			name: "successful generation",
			provider: NewAdvancedMockTTSProvider("test",
				WithAudioResponses([]byte("audio data"))),
			text:          "Hello world",
			expectedError: false,
		},
		{
			name: "error on generation",
			provider: NewAdvancedMockTTSProvider("test",
				WithError(NewTTSError("GenerateSpeech", ErrCodeNetworkError, nil))),
			text:          "Hello world",
			expectedError: true,
		},
		{
			name: "empty text",
			provider: NewAdvancedMockTTSProvider("test",
				WithAudioResponses([]byte("audio data"))),
			text:          "",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			audio, err := tt.provider.GenerateSpeech(ctx, tt.text)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, audio)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, audio)
			}
		})
	}
}

func TestTTSProvider_StreamGenerate(t *testing.T) {
	tests := []struct {
		name          string
		provider      iface.TTSProvider
		text          string
		expectedError bool
	}{
		{
			name: "successful streaming",
			provider: NewAdvancedMockTTSProvider("test",
				WithAudioResponses([]byte("streaming audio"))),
			text:          "Hello world",
			expectedError: false,
		},
		{
			name: "error on streaming",
			provider: NewAdvancedMockTTSProvider("test",
				WithError(NewTTSError("StreamGenerate", ErrCodeNetworkError, nil))),
			text:          "Hello world",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			reader, err := tt.provider.StreamGenerate(ctx, tt.text)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, reader)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, reader)

				// Read from stream
				buffer := make([]byte, 1024)
				n, readErr := reader.Read(buffer)
				assert.NoError(t, readErr)
				assert.Greater(t, n, 0)
			}
		})
	}
}

func TestTTSProvider_InterfaceCompliance(t *testing.T) {
	provider := NewAdvancedMockTTSProvider("test")
	AssertTTSProviderInterface(t, provider)
}

func TestStreamGenerate_Read(t *testing.T) {
	ctx := context.Background()
	provider := NewAdvancedMockTTSProvider("test",
		WithAudioResponses([]byte("test audio data")))
	reader, err := provider.StreamGenerate(ctx, "Test")
	require.NoError(t, err)
	require.NotNil(t, reader)

	buffer := make([]byte, 1024)
	n, err := reader.Read(buffer)
	assert.NoError(t, err)
	assert.Greater(t, n, 0)

	// Test EOF
	_, err = reader.Read(buffer)
	assert.Equal(t, io.EOF, err)
}
