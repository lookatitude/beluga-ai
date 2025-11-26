package rnnoise

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRNNoiseProvider(t *testing.T) {
	tests := []struct {
		config  *noise.Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &noise.Config{
				Provider:   "rnnoise",
				FrameSize:  480,
				SampleRate: 48000,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "invalid frame size",
			config: &noise.Config{
				Provider:   "rnnoise",
				FrameSize:  240,
				SampleRate: 48000,
			},
			wantErr: true,
		},
		{
			name: "invalid sample rate",
			config: &noise.Config{
				Provider:   "rnnoise",
				FrameSize:  480,
				SampleRate: 16000,
			},
			wantErr: true,
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
	config := &noise.Config{
		Provider:   "rnnoise",
		FrameSize:  480,
		SampleRate: 48000,
	}

	provider, err := NewRNNoiseProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	// Create audio frame of 480 samples * 2 bytes = 960 bytes
	audio := make([]byte, 960)

	processed, err := provider.Process(ctx, audio)
	require.NoError(t, err)
	assert.NotNil(t, processed)
}

func TestRNNoiseProvider_ProcessStream(t *testing.T) {
	config := &noise.Config{
		Provider:   "rnnoise",
		FrameSize:  480,
		SampleRate: 48000,
	}

	provider, err := NewRNNoiseProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audioCh := make(chan []byte, 1)

	audio := make([]byte, 960)
	audioCh <- audio
	close(audioCh)

	processedCh, err := provider.ProcessStream(ctx, audioCh)
	require.NoError(t, err)

	// Receive processed audio
	processed := <-processedCh
	assert.NotNil(t, processed)
}

func TestRNNoiseProvider_ProcessStream_ContextCancellation(t *testing.T) {
	config := &noise.Config{
		Provider:   "rnnoise",
		FrameSize:  480,
		SampleRate: 48000,
	}

	provider, err := NewRNNoiseProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	audioCh := make(chan []byte, 1)

	cancel()

	processedCh, err := provider.ProcessStream(ctx, audioCh)
	require.NoError(t, err)

	// Channel should be closed
	_, ok := <-processedCh
	assert.False(t, ok)
}

func TestRNNoiseProvider_Process_EmptyAudio(t *testing.T) {
	config := &noise.Config{
		Provider:   "rnnoise",
		FrameSize:  480,
		SampleRate: 48000,
	}

	provider, err := NewRNNoiseProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	processed, err := provider.Process(ctx, []byte{})
	require.NoError(t, err)
	assert.Empty(t, processed)
}

func TestDefaultRNNoiseConfig(t *testing.T) {
	config := DefaultRNNoiseConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 480, config.FrameSize)
	assert.Equal(t, 48000, config.SampleRate)
	assert.Equal(t, "rnnoise.rnn", config.ModelPath)
}

func TestRNNoiseProvider_Process_Padding(t *testing.T) {
	config := &noise.Config{
		Provider:   "rnnoise",
		FrameSize:  480,
		SampleRate: 48000,
	}

	provider, err := NewRNNoiseProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	// Audio smaller than expected size (should pad)
	audio := make([]byte, 480) // 240 samples * 2 bytes

	processed, err := provider.Process(ctx, audio)
	require.NoError(t, err)
	assert.NotNil(t, processed)
	assert.Len(t, processed, len(audio))
}

func TestRNNoiseProvider_Process_Truncation(t *testing.T) {
	config := &noise.Config{
		Provider:   "rnnoise",
		FrameSize:  480,
		SampleRate: 48000,
	}

	provider, err := NewRNNoiseProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	// Audio larger than expected size (should truncate)
	audio := make([]byte, 2000) // Larger than 960 bytes

	processed, err := provider.Process(ctx, audio)
	require.NoError(t, err)
	assert.NotNil(t, processed)
	assert.Len(t, processed, 960) // Should be truncated to expected size
}

func TestRNNoiseModel_IsLoaded(t *testing.T) {
	model := NewRNNoiseModel("test-model.rnn")
	assert.NotNil(t, model)
	assert.False(t, model.IsLoaded())

	// Load model
	err := model.Load()
	require.NoError(t, err)
	assert.True(t, model.IsLoaded())
}

func TestRNNoiseModel_Close(t *testing.T) {
	model := NewRNNoiseModel("test-model.rnn")
	assert.NotNil(t, model)

	// Load first
	err := model.Load()
	require.NoError(t, err)
	assert.True(t, model.IsLoaded())

	// Close model
	err = model.Close()
	require.NoError(t, err)
	assert.False(t, model.IsLoaded())
}
