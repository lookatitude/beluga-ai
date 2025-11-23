package vad

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	// Import providers to trigger init() registration
	_ "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/energy"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/rnnoise"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/silero"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/webrtc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVADProvider_Integration(t *testing.T) {
	// Integration test for VAD provider creation and basic operations
	// This test uses mock providers to avoid requiring real models

	t.Run("provider creation", func(t *testing.T) {
		config := vad.DefaultConfig()
		config.Provider = "energy"

		// Test that provider can be created via registry
		registry := vad.GetRegistry()
		provider, err := registry.GetProvider("energy", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("mock provider processing", func(t *testing.T) {
		mockProvider := vad.NewAdvancedMockVADProvider("test",
			vad.WithSpeechResults(true),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		hasVoice, err := mockProvider.Process(ctx, audio)
		require.NoError(t, err)
		assert.True(t, hasVoice)
	})

	t.Run("mock provider streaming", func(t *testing.T) {
		mockProvider := vad.NewAdvancedMockVADProvider("test",
			vad.WithSpeechResults(true, false, true),
			vad.WithProcessingDelay(10*time.Millisecond),
		)

		ctx := context.Background()
		audioCh := make(chan []byte, 10)
		resultCh, err := mockProvider.ProcessStream(ctx, audioCh)
		require.NoError(t, err)
		require.NotNil(t, resultCh)

		// Send audio chunks
		audioCh <- []byte{1, 2, 3, 4, 5}
		audioCh <- []byte{6, 7, 8, 9, 10}
		close(audioCh)

		// Receive results
		timeout := time.After(2 * time.Second)
		received := 0
		for {
			select {
			case result := <-resultCh:
				if result.Error == nil {
					received++
					if result.HasVoice {
						assert.Greater(t, result.Confidence, 0.0)
					}
				}
			case <-timeout:
				if received == 0 {
					t.Fatal("timeout waiting for results")
				}
				return
			}
		}
	})
}

func TestVADProvider_ErrorHandling(t *testing.T) {
	t.Run("timeout error retry", func(t *testing.T) {
		mockProvider := vad.NewAdvancedMockVADProvider("test",
			vad.WithError(vad.NewVADError("Process", vad.ErrCodeTimeout, nil)),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		_, err := mockProvider.Process(ctx, audio)
		assert.Error(t, err)
		assert.True(t, vad.IsRetryableError(err))
	})

	t.Run("invalid config error no retry", func(t *testing.T) {
		mockProvider := vad.NewAdvancedMockVADProvider("test",
			vad.WithError(vad.NewVADError("Process", vad.ErrCodeInvalidConfig, nil)),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		_, err := mockProvider.Process(ctx, audio)
		assert.Error(t, err)
		assert.False(t, vad.IsRetryableError(err))
	})
}

func TestVADProvider_ConcurrentRequests(t *testing.T) {
	mockProvider := vad.NewAdvancedMockVADProvider("test",
		vad.WithSpeechResults(true),
	)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	// Test concurrent processing
	const numGoroutines = 10
	results := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			hasVoice, err := mockProvider.Process(ctx, audio)
			if err != nil {
				errors <- err
			} else {
				results <- hasVoice
			}
		}()
	}

	// Collect results
	successCount := 0
	errorCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-results:
			successCount++
		case <-errors:
			errorCount++
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for results")
		}
	}

	assert.Equal(t, numGoroutines, successCount)
	assert.Equal(t, 0, errorCount)
}
