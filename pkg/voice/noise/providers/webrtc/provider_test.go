package webrtc

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebRTCNoiseProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *noise.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &noise.Config{
				Provider: "webrtc",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewWebRTCNoiseProvider(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestWebRTCNoiseProvider_Process(t *testing.T) {
	config := &noise.Config{
		Provider: "webrtc",
	}

	provider, err := NewWebRTCNoiseProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{128, 129, 130, 131, 132}

	processed, err := provider.Process(ctx, audio)
	assert.NoError(t, err)
	assert.NotNil(t, processed)
	assert.Equal(t, len(audio), len(processed))
}

func TestWebRTCNoiseProvider_ProcessStream(t *testing.T) {
	config := &noise.Config{
		Provider: "webrtc",
	}

	provider, err := NewWebRTCNoiseProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audioCh := make(chan []byte, 1)

	audio := []byte{128, 129, 130, 131, 132}
	audioCh <- audio
	close(audioCh)

	processedCh, err := provider.ProcessStream(ctx, audioCh)
	assert.NoError(t, err)

	// Receive processed audio
	processed := <-processedCh
	assert.NotNil(t, processed)
}

func TestWebRTCNoiseProvider_ProcessStream_ContextCancellation(t *testing.T) {
	config := &noise.Config{
		Provider: "webrtc",
	}

	provider, err := NewWebRTCNoiseProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	audioCh := make(chan []byte, 1)

	cancel()

	processedCh, err := provider.ProcessStream(ctx, audioCh)
	assert.NoError(t, err)

	// Channel should be closed
	_, ok := <-processedCh
	assert.False(t, ok)
}

func TestWebRTCNoiseProvider_Process_EmptyAudio(t *testing.T) {
	config := &noise.Config{
		Provider: "webrtc",
	}

	provider, err := NewWebRTCNoiseProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	processed, err := provider.Process(ctx, []byte{})
	assert.NoError(t, err)
	assert.Empty(t, processed)
}

func TestDefaultWebRTCNoiseConfig(t *testing.T) {
	config := DefaultWebRTCNoiseConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 2, config.Aggressiveness)
	assert.True(t, config.EnableHighPassFilter)
	assert.False(t, config.EnableEchoCancellation)
}
