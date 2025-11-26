package tts

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	// Import providers to trigger init() registration.
	_ "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/azure"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/elevenlabs"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/google"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTTSProvider_Integration(t *testing.T) {
	// Integration test for TTS provider creation and basic operations
	// This test uses mock providers to avoid requiring real API keys

	t.Run("provider creation", func(t *testing.T) {
		config := tts.DefaultConfig()
		config.Provider = "openai"
		config.APIKey = "test-key"

		// Test that provider can be created via registry
		registry := tts.GetRegistry()
		provider, err := registry.GetProvider("openai", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("mock provider generation", func(t *testing.T) {
		mockProvider := tts.NewAdvancedMockTTSProvider("test",
			tts.WithAudioResponses([]byte("Integration test audio")),
		)

		ctx := context.Background()
		text := "Integration test text"

		audio, err := mockProvider.GenerateSpeech(ctx, text)
		require.NoError(t, err)
		assert.NotNil(t, audio)
		assert.NotEmpty(t, audio)
	})

	t.Run("mock provider streaming", func(t *testing.T) {
		mockProvider := tts.NewAdvancedMockTTSProvider("test",
			tts.WithAudioResponses([]byte("Streaming audio data")),
			tts.WithStreamingDelay(10*time.Millisecond),
		)

		ctx := context.Background()
		text := "Streaming test text"

		reader, err := mockProvider.StreamGenerate(ctx, text)
		require.NoError(t, err)
		require.NotNil(t, reader)

		// Read from stream
		buffer := make([]byte, 1024)
		n, err := reader.Read(buffer)
		require.NoError(t, err)
		assert.Positive(t, n)
	})
}

func TestTTSProvider_ErrorHandling(t *testing.T) {
	t.Run("network error retry", func(t *testing.T) {
		mockProvider := tts.NewAdvancedMockTTSProvider("test",
			tts.WithError(tts.NewTTSError("GenerateSpeech", tts.ErrCodeNetworkError, nil)),
		)

		ctx := context.Background()
		text := "Test text"

		_, err := mockProvider.GenerateSpeech(ctx, text)
		require.Error(t, err)
		assert.True(t, tts.IsRetryableError(err))
	})

	t.Run("authentication error no retry", func(t *testing.T) {
		mockProvider := tts.NewAdvancedMockTTSProvider("test",
			tts.WithError(tts.NewTTSError("GenerateSpeech", tts.ErrCodeAuthentication, nil)),
		)

		ctx := context.Background()
		text := "Test text"

		_, err := mockProvider.GenerateSpeech(ctx, text)
		require.Error(t, err)
		assert.False(t, tts.IsRetryableError(err))
	})
}

func TestTTSProvider_ConcurrentRequests(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test",
		tts.WithAudioResponses([]byte("Concurrent test audio")),
	)

	ctx := context.Background()
	text := "Concurrent test text"

	// Test concurrent generations
	const numGoroutines = 10
	results := make(chan []byte, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			audio, err := mockProvider.GenerateSpeech(ctx, text)
			if err != nil {
				errors <- err
			} else {
				results <- audio
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

func TestTTSProvider_StreamingRead(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test",
		tts.WithAudioResponses([]byte("test audio data for streaming")),
		tts.WithStreamingDelay(1*time.Millisecond),
	)

	ctx := context.Background()
	reader, err := mockProvider.StreamGenerate(ctx, "Test")
	require.NoError(t, err)
	require.NotNil(t, reader)

	// Read all data
	buffer := make([]byte, 1024)
	totalRead := 0
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		totalRead += n
	}

	assert.Positive(t, totalRead)
}
