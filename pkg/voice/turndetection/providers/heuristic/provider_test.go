package heuristic

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHeuristicProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *turndetection.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &turndetection.Config{
				Provider: "heuristic",
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
			config: &turndetection.Config{
				Provider: "heuristic",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewHeuristicProvider(tt.config)
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

func TestHeuristicProvider_DetectTurn(t *testing.T) {
	config := &turndetection.Config{
		Provider: "heuristic",
	}

	provider, err := NewHeuristicProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	// Test detection
	turn, err := provider.DetectTurn(ctx, audio)
	assert.NoError(t, err)
	// Result depends on implementation, but should not error
	_ = turn
}

func TestHeuristicProvider_DetectTurnWithSilence(t *testing.T) {
	config := &turndetection.Config{
		Provider: "heuristic",
	}

	provider, err := NewHeuristicProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	// Test with silence duration below threshold
	turn, err := provider.DetectTurnWithSilence(ctx, audio, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.False(t, turn)

	// Test with silence duration above threshold
	turn, err = provider.DetectTurnWithSilence(ctx, audio, 600*time.Millisecond)
	assert.NoError(t, err)
	assert.True(t, turn)
}

func TestHeuristicProvider_detectTurnFromTranscript(t *testing.T) {
	config := &turndetection.Config{
		Provider: "heuristic",
	}

	provider, err := NewHeuristicProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	heuristicProvider := provider.(*HeuristicProvider)

	// Test with sentence ending
	turn := heuristicProvider.detectTurnFromTranscript("Hello world.", true)
	assert.True(t, turn)

	// Test with question
	turn = heuristicProvider.detectTurnFromTranscript("What is this?", true)
	assert.True(t, turn)

	// Test with short transcript
	turn = heuristicProvider.detectTurnFromTranscript("Hi", true)
	assert.False(t, turn)

	// Test with incomplete transcript
	turn = heuristicProvider.detectTurnFromTranscript("Hello world", false)
	assert.False(t, turn)
}

func TestDefaultHeuristicConfig(t *testing.T) {
	config := DefaultHeuristicConfig()
	assert.NotNil(t, config)
	assert.Equal(t, ".!?", config.SentenceEndMarkers)
	assert.Equal(t, 500*time.Millisecond, config.MinSilenceDuration)
	assert.Equal(t, 10, config.MinTurnLength)
}
