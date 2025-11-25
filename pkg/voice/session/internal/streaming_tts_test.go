package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamingTTS(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test")
	stts := NewStreamingTTS(mockProvider)
	assert.NotNil(t, stts)
	assert.False(t, stts.IsActive())
}

func TestNewStreamingTTS_NilProvider(t *testing.T) {
	stts := NewStreamingTTS(nil)
	assert.NotNil(t, stts)
	assert.False(t, stts.IsActive())
}

func TestStreamingTTS_StartStream_Success(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test")
	stts := NewStreamingTTS(mockProvider)

	ctx := context.Background()
	reader, err := stts.StartStream(ctx, "Hello, world!")
	assert.NoError(t, err)
	assert.NotNil(t, reader)
	assert.True(t, stts.IsActive())
	assert.Equal(t, reader, stts.GetCurrentReader())
}

func TestStreamingTTS_StartStream_NilProvider(t *testing.T) {
	stts := NewStreamingTTS(nil)

	ctx := context.Background()
	reader, err := stts.StartStream(ctx, "Hello, world!")
	assert.Error(t, err)
	assert.Nil(t, reader)
	assert.Contains(t, err.Error(), "not set")
}

func TestStreamingTTS_StartStream_ProviderError(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test", tts.WithError(errors.New("TTS error")))
	stts := NewStreamingTTS(mockProvider)

	ctx := context.Background()
	reader, err := stts.StartStream(ctx, "Hello, world!")
	assert.Error(t, err)
	assert.Nil(t, reader)
	assert.Contains(t, err.Error(), "failed to start streaming")
}

func TestStreamingTTS_Stop(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test")
	stts := NewStreamingTTS(mockProvider)

	ctx := context.Background()
	reader, err := stts.StartStream(ctx, "Hello, world!")
	require.NoError(t, err)
	assert.True(t, stts.IsActive())
	assert.NotNil(t, reader)

	stts.Stop()
	assert.False(t, stts.IsActive())
	assert.Nil(t, stts.GetCurrentReader())
}

func TestStreamingTTS_IsActive(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test")
	stts := NewStreamingTTS(mockProvider)

	assert.False(t, stts.IsActive())

	ctx := context.Background()
	_, err := stts.StartStream(ctx, "Hello")
	require.NoError(t, err)
	assert.True(t, stts.IsActive())

	stts.Stop()
	assert.False(t, stts.IsActive())
}

func TestStreamingTTS_GetCurrentReader(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test")
	stts := NewStreamingTTS(mockProvider)

	// Initially nil
	assert.Nil(t, stts.GetCurrentReader())

	ctx := context.Background()
	reader, err := stts.StartStream(ctx, "Hello, world!")
	require.NoError(t, err)
	assert.Equal(t, reader, stts.GetCurrentReader())

	// Read from reader to verify it works
	data := make([]byte, 100)
	n, err := reader.Read(data)
	assert.NoError(t, err)
	assert.Greater(t, n, 0)

	stts.Stop()
	assert.Nil(t, stts.GetCurrentReader())
}

func TestStreamingTTS_ConcurrentAccess(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test")
	stts := NewStreamingTTS(mockProvider)

	ctx := context.Background()
	_, err := stts.StartStream(ctx, "Hello")
	require.NoError(t, err)

	// Concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			stts.IsActive()
			stts.GetCurrentReader()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
