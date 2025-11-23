package spectral

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSpectralProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *noise.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &noise.Config{
				Provider: "spectral",
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
			provider, err := NewSpectralProvider(tt.config)
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

func TestSpectralProvider_Process(t *testing.T) {
	config := &noise.Config{
		Provider: "spectral",
	}

	provider, err := NewSpectralProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{128, 129, 130, 131, 132}

	processed, err := provider.Process(ctx, audio)
	assert.NoError(t, err)
	assert.NotNil(t, processed)
	assert.Equal(t, len(audio), len(processed))
}

func TestSpectralProvider_ProcessStream(t *testing.T) {
	config := &noise.Config{
		Provider: "spectral",
	}

	provider, err := NewSpectralProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audioCh := make(chan []byte, 2)
	audioCh <- []byte{128, 129, 130}
	audioCh <- []byte{131, 132, 133}
	close(audioCh)

	processedCh, err := provider.ProcessStream(ctx, audioCh)
	assert.NoError(t, err)
	assert.NotNil(t, processedCh)

	// Receive processed audio
	processed1 := <-processedCh
	assert.NotNil(t, processed1)

	processed2 := <-processedCh
	assert.NotNil(t, processed2)
}

func TestSpectralProvider_ProcessStream_ContextCancellation(t *testing.T) {
	config := &noise.Config{
		Provider: "spectral",
	}

	provider, err := NewSpectralProvider(config)
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

func TestSpectralProvider_Process_EmptyAudio(t *testing.T) {
	config := &noise.Config{
		Provider: "spectral",
	}

	provider, err := NewSpectralProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	processed, err := provider.Process(ctx, []byte{})
	assert.NoError(t, err)
	assert.Empty(t, processed)
}

func TestDefaultSpectralConfig(t *testing.T) {
	config := DefaultSpectralConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 2.0, config.Alpha)
	assert.Equal(t, 0.1, config.Beta)
	assert.Equal(t, 512, config.FFTSize)
	assert.Equal(t, "hann", config.WindowType)
}
