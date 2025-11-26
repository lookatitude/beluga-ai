package internal

import (
	"context"
	"testing"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVoiceSessionImpl_ProcessAudio_NotActive(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	err := impl.ProcessAudio(ctx, audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_active")
}

func TestVoiceSessionImpl_ProcessAudio_Active(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	err := impl.Start(ctx)
	require.NoError(t, err)

	audio := []byte{1, 2, 3, 4, 5}
	err = impl.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Should transition to processing state
	assert.Equal(t, sessioniface.SessionState("processing"), impl.GetState())
}

func TestVoiceSessionImpl_ProcessAudio_WithNoiseCancellation(t *testing.T) {
	impl := createTestSessionImpl(t)

	// Create a mock noise cancellation provider
	mockNoise := &mockNoiseCancellation{
		processFunc: func(ctx context.Context, audio []byte) ([]byte, error) {
			// Return cleaned audio
			return []byte{5, 4, 3, 2, 1}, nil
		},
	}

	impl.noiseCancellation = mockNoise

	ctx := context.Background()
	err := impl.Start(ctx)
	require.NoError(t, err)

	audio := []byte{1, 2, 3, 4, 5}
	err = impl.ProcessAudio(ctx, audio)
	require.NoError(t, err)
}

func TestVoiceSessionImpl_ProcessAudio_NoiseCancellationError(t *testing.T) {
	impl := createTestSessionImpl(t)

	// Create a mock noise cancellation that returns error
	mockNoise := &mockNoiseCancellation{
		processFunc: func(ctx context.Context, audio []byte) ([]byte, error) {
			return nil, assert.AnError
		},
	}

	impl.noiseCancellation = mockNoise

	ctx := context.Background()
	err := impl.Start(ctx)
	require.NoError(t, err)

	audio := []byte{1, 2, 3, 4, 5}
	// Should continue processing even if noise cancellation fails
	err = impl.ProcessAudio(ctx, audio)
	require.NoError(t, err)
}

// Mock noise cancellation for testing.
type mockNoiseCancellation struct {
	processFunc func(ctx context.Context, audio []byte) ([]byte, error)
}

func (m *mockNoiseCancellation) Process(ctx context.Context, audio []byte) ([]byte, error) {
	if m.processFunc != nil {
		return m.processFunc(ctx, audio)
	}
	return audio, nil
}

func (m *mockNoiseCancellation) ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan []byte, error) {
	output := make(chan []byte, 10)

	go func() {
		defer close(output)

		for data := range audioCh {
			if m.processFunc != nil {
				processed, err := m.processFunc(ctx, data)
				if err != nil {
					return
				}
				output <- processed
			} else {
				output <- data
			}
		}
	}()

	return output, nil
}
