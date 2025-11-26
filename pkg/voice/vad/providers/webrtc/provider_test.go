package webrtc

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebRTCProvider(t *testing.T) {
	tests := []struct {
		config  *vad.Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &vad.Config{
				Provider: "webrtc",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "config with defaults",
			config: &vad.Config{
				Provider: "webrtc",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewWebRTCProvider(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestWebRTCProvider_Process(t *testing.T) {
	config := &vad.Config{
		Provider: "webrtc",
	}

	provider, err := NewWebRTCProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := make([]byte, 640) // 320 samples * 2 bytes

	// Test processing
	speech, err := provider.Process(ctx, audio)
	require.NoError(t, err)
	// Result depends on audio content, but should not error
	_ = speech
}

func TestDefaultWebRTCConfig(t *testing.T) {
	config := DefaultWebRTCConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 0, config.Mode)
	assert.Equal(t, 16000, config.SampleRate)
	assert.Equal(t, 320, config.FrameSize)
}

func TestGetThresholdForMode(t *testing.T) {
	thresholds := []float64{0.01, 0.015, 0.02, 0.025}
	for mode := 0; mode < 4; mode++ {
		threshold := getThresholdForMode(mode)
		assert.Equal(t, thresholds[mode], threshold)
	}

	// Test invalid mode
	threshold := getThresholdForMode(10)
	assert.Equal(t, thresholds[0], threshold)
}

func TestWebRTCProvider_ProcessStream(t *testing.T) {
	config := &vad.Config{
		Provider: "webrtc",
	}

	provider, err := NewWebRTCProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audioCh := make(chan []byte, 1)

	audio := make([]byte, 640)
	audioCh <- audio
	close(audioCh)

	resultCh, err := provider.ProcessStream(ctx, audioCh)
	require.NoError(t, err)

	// Read results
	result := <-resultCh
	assert.NotNil(t, result)
}

func TestWebRTCProvider_ProcessStream_ContextCancellation(t *testing.T) {
	config := &vad.Config{
		Provider: "webrtc",
	}

	provider, err := NewWebRTCProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	audioCh := make(chan []byte, 1)

	cancel()

	resultCh, err := provider.ProcessStream(ctx, audioCh)
	require.NoError(t, err)

	// Channel should be closed
	_, ok := <-resultCh
	assert.False(t, ok)
}

func TestWebRTCProvider_Process_EmptyAudio(t *testing.T) {
	config := &vad.Config{
		Provider: "webrtc",
	}

	provider, err := NewWebRTCProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	// Empty audio should return an error for WebRTC (requires minimum frame size)
	_, err = provider.Process(ctx, []byte{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "frame_size_error")
}
