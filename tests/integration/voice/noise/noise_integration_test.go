package noise

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
	// Import providers to trigger init() registration.
	_ "github.com/lookatitude/beluga-ai/pkg/voice/noise/providers/rnnoise"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/noise/providers/spectral"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/noise/providers/webrtc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoiseCancellation_Integration(t *testing.T) {
	// Integration test for Noise Cancellation provider creation and basic operations
	// This test uses mock providers to avoid requiring real models

	t.Run("provider creation", func(t *testing.T) {
		config := noise.DefaultConfig()
		config.Provider = "spectral"

		// Test that provider can be created via registry
		registry := noise.GetRegistry()
		provider, err := registry.GetProvider("spectral", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("mock noise cancellation process", func(t *testing.T) {
		mockNoise := noise.NewAdvancedMockNoiseCancellation("test",
			noise.WithProcessedAudio([]byte{5, 4, 3, 2, 1}),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		cleaned, err := mockNoise.Process(ctx, audio)
		require.NoError(t, err)
		assert.NotNil(t, cleaned)
	})

	t.Run("mock noise cancellation stream", func(t *testing.T) {
		mockNoise := noise.NewAdvancedMockNoiseCancellation("test",
			noise.WithProcessedAudio([]byte{5, 4, 3}, []byte{2, 1}),
			noise.WithProcessingDelay(10*time.Millisecond),
		)

		ctx := context.Background()
		audioCh := make(chan []byte, 2)
		audioCh <- []byte{1, 2, 3}
		audioCh <- []byte{4, 5}
		close(audioCh)

		cleanedCh, err := mockNoise.ProcessStream(ctx, audioCh)
		require.NoError(t, err)
		assert.NotNil(t, cleanedCh)

		// Receive cleaned audio
		timeout := time.After(2 * time.Second)
		received := 0
		for {
			select {
			case cleaned, ok := <-cleanedCh:
				if !ok {
					if received == 0 {
						t.Fatal("expected data but received none")
					}
					return
				}
				assert.NotNil(t, cleaned)
				received++
			case <-timeout:
				if received == 0 {
					t.Fatal("timeout waiting for cleaned audio")
				}
				return
			}
		}
	})
}

func TestNoiseCancellation_ErrorHandling(t *testing.T) {
	t.Run("processing error", func(t *testing.T) {
		mockNoise := noise.NewAdvancedMockNoiseCancellation("test",
			noise.WithError(noise.NewNoiseCancellationError("Process", noise.ErrCodeProcessingError, nil)),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		cleaned, err := mockNoise.Process(ctx, audio)
		require.Error(t, err)
		assert.Nil(t, cleaned)
		assert.True(t, noise.IsRetryableError(err))
	})

	t.Run("timeout error retry", func(t *testing.T) {
		mockNoise := noise.NewAdvancedMockNoiseCancellation("test",
			noise.WithError(noise.NewNoiseCancellationError("Process", noise.ErrCodeTimeout, nil)),
		)

		ctx := context.Background()
		_, err := mockNoise.Process(ctx, []byte{1, 2, 3})
		require.Error(t, err)
		assert.True(t, noise.IsRetryableError(err))
	})
}

func TestNoiseCancellation_ConcurrentOperations(t *testing.T) {
	mockNoise := noise.NewAdvancedMockNoiseCancellation("test")

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	// Test concurrent processing
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := mockNoise.Process(ctx, audio)
			if err != nil {
				errors <- err
			} else {
				done <- true
			}
		}()
	}

	// Collect results
	successCount := 0
	errorCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
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
