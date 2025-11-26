package stt

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSTTProvider_Transcribe(t *testing.T) {
	tests := []struct {
		provider      iface.STTProvider
		name          string
		expectedText  string
		audio         []byte
		expectedError bool
	}{
		{
			name: "successful transcription",
			provider: NewAdvancedMockSTTProvider("test",
				WithTranscriptions("Hello world")),
			audio:         []byte{1, 2, 3, 4, 5},
			expectedText:  "Hello world",
			expectedError: false,
		},
		{
			name: "error on transcription",
			provider: NewAdvancedMockSTTProvider("test",
				WithError(NewSTTError("Transcribe", ErrCodeNetworkError, nil))),
			audio:         []byte{1, 2, 3, 4, 5},
			expectedText:  "",
			expectedError: true,
		},
		{
			name: "empty audio",
			provider: NewAdvancedMockSTTProvider("test",
				WithTranscriptions("Empty audio")),
			audio:         []byte{},
			expectedText:  "Empty audio",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			text, err := tt.provider.Transcribe(ctx, tt.audio)

			if tt.expectedError {
				require.Error(t, err)
				assert.Empty(t, text)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedText, text)
			}
		})
	}
}

func TestSTTProvider_StartStreaming(t *testing.T) {
	tests := []struct {
		provider      iface.STTProvider
		name          string
		expectedError bool
	}{
		{
			name: "successful streaming start",
			provider: NewAdvancedMockSTTProvider("test",
				WithTranscriptions("Hello", "world")),
			expectedError: false,
		},
		{
			name: "error on streaming start",
			provider: NewAdvancedMockSTTProvider("test",
				WithError(NewSTTError("StartStreaming", ErrCodeNetworkError, nil))),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			session, err := tt.provider.StartStreaming(ctx)

			if tt.expectedError {
				require.Error(t, err)
				assert.Nil(t, session)
			} else {
				require.NoError(t, err)
				require.NotNil(t, session)
				defer func() { _ = session.Close() }()

				// Test receiving transcripts
				timeout := time.After(2 * time.Second)
				select {
				case result := <-session.ReceiveTranscript():
					assert.NotEmpty(t, result.Text)
					require.NoError(t, result.Error)
				case <-timeout:
					t.Fatal("timeout waiting for transcript")
				}
			}
		})
	}
}

func TestSTTProvider_InterfaceCompliance(t *testing.T) {
	provider := NewAdvancedMockSTTProvider("test")
	AssertSTTProviderInterface(t, provider)
}

func TestStreamingSession_SendAudio(t *testing.T) {
	ctx := context.Background()
	provider := NewAdvancedMockSTTProvider("test",
		WithTranscriptions("Test"))
	session, err := provider.StartStreaming(ctx)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer func() { _ = session.Close() }()

	err = session.SendAudio(ctx, []byte{1, 2, 3, 4, 5})
	require.NoError(t, err)
}

func TestStreamingSession_Close(t *testing.T) {
	ctx := context.Background()
	provider := NewAdvancedMockSTTProvider("test",
		WithTranscriptions("Test"))
	session, err := provider.StartStreaming(ctx)
	require.NoError(t, err)
	require.NotNil(t, session)

	err = session.Close()
	require.NoError(t, err)

	// Closing again should not error
	err = session.Close()
	require.NoError(t, err)
}
