package turndetection

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
	// Import providers to trigger init() registration.
	_ "github.com/lookatitude/beluga-ai/pkg/voice/turndetection/providers/heuristic"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/turndetection/providers/onnx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTurnDetector_Integration(t *testing.T) {
	// Integration test for Turn Detection provider creation and basic operations
	// This test uses mock providers to avoid requiring real models

	t.Run("provider creation", func(t *testing.T) {
		config := turndetection.DefaultConfig()
		config.Provider = "heuristic"

		// Test that provider can be created via registry
		registry := turndetection.GetRegistry()
		provider, err := registry.GetProvider("heuristic", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("mock provider detection", func(t *testing.T) {
		mockProvider := turndetection.NewAdvancedMockTurnDetector("test",
			turndetection.WithTurnResults(true),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		turnEnd, err := mockProvider.DetectTurn(ctx, audio)
		require.NoError(t, err)
		assert.True(t, turnEnd)
	})

	t.Run("mock provider silence detection", func(t *testing.T) {
		mockProvider := turndetection.NewAdvancedMockTurnDetector("test",
			turndetection.WithTurnResults(true),
			turndetection.WithProcessingDelay(10*time.Millisecond),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}
		silenceDuration := 500 * time.Millisecond

		turnEnd, err := mockProvider.DetectTurnWithSilence(ctx, audio, silenceDuration)
		require.NoError(t, err)
		assert.True(t, turnEnd)
	})
}

func TestTurnDetector_ErrorHandling(t *testing.T) {
	t.Run("timeout error retry", func(t *testing.T) {
		mockProvider := turndetection.NewAdvancedMockTurnDetector("test",
			turndetection.WithError(turndetection.NewTurnDetectionError("DetectTurn", turndetection.ErrCodeTimeout, nil)),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		_, err := mockProvider.DetectTurn(ctx, audio)
		require.Error(t, err)
		assert.True(t, turndetection.IsRetryableError(err))
	})

	t.Run("invalid config error no retry", func(t *testing.T) {
		mockProvider := turndetection.NewAdvancedMockTurnDetector("test",
			turndetection.WithError(turndetection.NewTurnDetectionError("DetectTurn", turndetection.ErrCodeInvalidConfig, nil)),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		_, err := mockProvider.DetectTurn(ctx, audio)
		require.Error(t, err)
		assert.False(t, turndetection.IsRetryableError(err))
	})
}

func TestTurnDetector_ConcurrentRequests(t *testing.T) {
	mockProvider := turndetection.NewAdvancedMockTurnDetector("test",
		turndetection.WithTurnResults(true),
	)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	// Test concurrent detections
	const numGoroutines = 10
	results := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			turnEnd, err := mockProvider.DetectTurn(ctx, audio)
			if err != nil {
				errors <- err
			} else {
				results <- turnEnd
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
