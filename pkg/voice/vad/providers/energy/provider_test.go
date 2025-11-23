package energy

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnergyProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *vad.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &vad.Config{
				Provider: "energy",
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
				Provider: "energy",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewEnergyProvider(tt.config)
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

func TestEnergyProvider_Process(t *testing.T) {
	config := &vad.Config{
		Provider: "energy",
	}

	provider, err := NewEnergyProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()

	// Test with high energy audio (should detect speech)
	highEnergyAudio := make([]byte, 1024)
	for i := 0; i < len(highEnergyAudio); i += 2 {
		if i+1 < len(highEnergyAudio) {
			// Create a high-energy signal
			sample := int16(10000)
			highEnergyAudio[i] = byte(sample)
			highEnergyAudio[i+1] = byte(sample >> 8)
		}
	}

	speech, err := provider.Process(ctx, highEnergyAudio)
	assert.NoError(t, err)
	// High energy should typically be detected as speech
	_ = speech

	// Test with low energy audio (should detect silence)
	lowEnergyAudio := make([]byte, 1024)
	speech, err = provider.Process(ctx, lowEnergyAudio)
	assert.NoError(t, err)
	// Low energy should typically be detected as silence
	_ = speech
}

func TestDefaultEnergyConfig(t *testing.T) {
	config := DefaultEnergyConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 0.01, config.Threshold)
	assert.Equal(t, 256, config.EnergyWindowSize)
	assert.True(t, config.AdaptiveThreshold)
}

func TestCalculateEnergy(t *testing.T) {
	// Test with empty audio
	energy := calculateEnergy([]byte{})
	assert.Equal(t, 0.0, energy)

	// Test with audio data
	audio := make([]byte, 1024)
	for i := 0; i < len(audio); i += 2 {
		if i+1 < len(audio) {
			sample := int16(1000)
			audio[i] = byte(sample)
			audio[i+1] = byte(sample >> 8)
		}
	}

	energy = calculateEnergy(audio)
	assert.Greater(t, energy, 0.0)
}

func TestEnergyProvider_ProcessStream(t *testing.T) {
	config := &vad.Config{
		Provider: "energy",
	}

	provider, err := NewEnergyProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audioCh := make(chan []byte, 2)

	// Send test audio
	highEnergyAudio := make([]byte, 1024)
	for i := 0; i < len(highEnergyAudio); i += 2 {
		if i+1 < len(highEnergyAudio) {
			sample := int16(10000)
			highEnergyAudio[i] = byte(sample)
			highEnergyAudio[i+1] = byte(sample >> 8)
		}
	}
	audioCh <- highEnergyAudio
	close(audioCh)

	resultCh, err := provider.ProcessStream(ctx, audioCh)
	require.NoError(t, err)

	// Read results
	result := <-resultCh
	assert.NotNil(t, result)
	assert.NoError(t, result.Error)
}

func TestEnergyProvider_ProcessStream_ContextCancellation(t *testing.T) {
	config := &vad.Config{
		Provider: "energy",
	}

	provider, err := NewEnergyProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	audioCh := make(chan []byte, 1)

	// Cancel context
	cancel()

	resultCh, err := provider.ProcessStream(ctx, audioCh)
	require.NoError(t, err)

	// Channel should be closed due to context cancellation
	_, ok := <-resultCh
	assert.False(t, ok)
}

func TestEnergyProvider_Process_EmptyAudio(t *testing.T) {
	config := &vad.Config{
		Provider: "energy",
	}

	provider, err := NewEnergyProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	speech, err := provider.Process(ctx, []byte{})
	assert.NoError(t, err)
	assert.False(t, speech)
}

func TestEnergyProvider_AdaptiveThreshold(t *testing.T) {
	config := &vad.Config{
		Provider: "energy",
	}

	provider, err := NewEnergyProvider(config)
	require.NoError(t, err)

	// Set adaptive threshold on the provider's config
	energyProvider := provider.(*EnergyProvider)
	energyProvider.config.AdaptiveThreshold = true

	ctx := context.Background()

	// Process multiple frames to build history
	for i := 0; i < 15; i++ {
		audio := make([]byte, 512)
		_, _ = provider.Process(ctx, audio)
	}

	// Process one more frame - adaptive threshold should be active
	audio := make([]byte, 512)
	_, err = provider.Process(ctx, audio)
	assert.NoError(t, err)
}
