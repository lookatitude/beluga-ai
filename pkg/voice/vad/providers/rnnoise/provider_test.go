package rnnoise

import (
	"context"
	"os"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRNNoiseProvider(t *testing.T) {
	tests := []struct {
		config  *vad.Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &vad.Config{
				Provider: "rnnoise",
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
				Provider: "rnnoise",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewRNNoiseProvider(tt.config)
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

func TestRNNoiseProvider_Process(t *testing.T) {
	// Create a temporary model file for testing
	tmpFile, err := os.CreateTemp("", "test_model.rnn")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config := &vad.Config{
		Provider: "rnnoise",
	}
	rnnoiseConfig := &RNNoiseConfig{
		Config:     config,
		ModelPath:  tmpFile.Name(),
		Threshold:  0.5,
		SampleRate: 48000,
		FrameSize:  480,
	}

	provider, err := NewRNNoiseProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Update provider config for test
	rnnoiseProvider := provider.(*RNNoiseProvider)
	rnnoiseProvider.config = rnnoiseConfig

	ctx := context.Background()
	audio := make([]byte, 960) // 480 samples * 2 bytes

	// Test processing
	speech, err := rnnoiseProvider.Process(ctx, audio)
	require.NoError(t, err)
	// Result depends on audio content, but should not error
	_ = speech
}

func TestRNNoiseProvider_ProcessStream(t *testing.T) {
	config := &vad.Config{
		Provider: "rnnoise",
	}

	provider, err := NewRNNoiseProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audioCh := make(chan []byte, 1)

	// Create audio frame of correct size
	audio := make([]byte, 960) // 480 samples * 2 bytes
	audioCh <- audio
	close(audioCh)

	resultCh, err := provider.ProcessStream(ctx, audioCh)
	require.NoError(t, err)

	// Read results
	result := <-resultCh
	assert.NotNil(t, result)
}

func TestRNNoiseProvider_ProcessStream_ContextCancellation(t *testing.T) {
	config := &vad.Config{
		Provider: "rnnoise",
	}

	provider, err := NewRNNoiseProvider(config)
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

func TestRNNoiseProvider_Process_InvalidFrameSize(t *testing.T) {
	config := &vad.Config{
		Provider: "rnnoise",
	}

	provider, err := NewRNNoiseProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	// Audio too small
	audio := make([]byte, 100)

	_, err = provider.Process(ctx, audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "frame_size_error")
}

func TestRNNoiseProvider_Process_EmptyAudio(t *testing.T) {
	config := &vad.Config{
		Provider: "rnnoise",
	}

	provider, err := NewRNNoiseProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = provider.Process(ctx, []byte{})
	require.Error(t, err)
}

func TestDefaultRNNoiseConfig(t *testing.T) {
	config := DefaultRNNoiseConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "rnnoise_model.rnn", config.ModelPath)
	assert.InEpsilon(t, 0.5, config.Threshold, 0.0001)
	assert.Equal(t, 48000, config.SampleRate)
	assert.Equal(t, 480, config.FrameSize)
}
