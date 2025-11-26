package stt

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/azure"    // Register azure provider
	_ "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram" // Register deepgram provider
	_ "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/google"   // Register google provider
	_ "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/openai"   // Register openai provider
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSTTProvider_Integration(t *testing.T) {
	// Integration test for STT provider creation and basic operations
	// This test uses mock providers to avoid requiring real API keys

	t.Run("provider creation", func(t *testing.T) {
		config := stt.DefaultConfig()
		config.Provider = "deepgram"
		config.APIKey = "test-key"

		// Test that provider can be created via registry
		registry := stt.GetRegistry()
		provider, err := registry.GetProvider("deepgram", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("mock provider transcription", func(t *testing.T) {
		mockProvider := stt.NewAdvancedMockSTTProvider("test",
			stt.WithTranscriptions("Integration test transcription"),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		text, err := mockProvider.Transcribe(ctx, audio)
		require.NoError(t, err)
		assert.Equal(t, "Integration test transcription", text)
	})

	t.Run("mock provider streaming", func(t *testing.T) {
		mockProvider := stt.NewAdvancedMockSTTProvider("test",
			stt.WithTranscriptions("Streaming", "test", "transcription"),
			stt.WithStreamingDelay(10*time.Millisecond),
		)

		ctx := context.Background()
		session, err := mockProvider.StartStreaming(ctx)
		require.NoError(t, err)
		require.NotNil(t, session)
		defer session.Close()

		// Send audio
		audio := []byte{1, 2, 3, 4, 5}
		err = session.SendAudio(ctx, audio)
		require.NoError(t, err)

		// Receive transcripts
		timeout := time.After(2 * time.Second)
		received := false
		for {
			select {
			case result := <-session.ReceiveTranscript():
				if result.Error == nil {
					assert.NotEmpty(t, result.Text)
					received = true
					if result.IsFinal {
						return
					}
				}
			case <-timeout:
				if !received {
					t.Fatal("timeout waiting for transcript")
				}
				return
			}
		}
	})
}

func TestSTTProvider_ErrorHandling(t *testing.T) {
	t.Run("network error retry", func(t *testing.T) {
		mockProvider := stt.NewAdvancedMockSTTProvider("test",
			stt.WithError(stt.NewSTTError("Transcribe", stt.ErrCodeNetworkError, nil)),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		_, err := mockProvider.Transcribe(ctx, audio)
		require.Error(t, err)
		assert.True(t, stt.IsRetryableError(err))
	})

	t.Run("authentication error no retry", func(t *testing.T) {
		mockProvider := stt.NewAdvancedMockSTTProvider("test",
			stt.WithError(stt.NewSTTError("Transcribe", stt.ErrCodeAuthentication, nil)),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		_, err := mockProvider.Transcribe(ctx, audio)
		require.Error(t, err)
		assert.False(t, stt.IsRetryableError(err))
	})
}

func TestSTTProvider_ConcurrentRequests(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test",
		stt.WithTranscriptions("Concurrent test"),
	)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	// Test concurrent transcriptions
	const numGoroutines = 10
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			text, err := mockProvider.Transcribe(ctx, audio)
			if err != nil {
				errors <- err
			} else {
				results <- text
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
