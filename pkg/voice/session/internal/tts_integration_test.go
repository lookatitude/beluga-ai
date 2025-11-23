package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	"github.com/stretchr/testify/assert"
)

func TestNewTTSIntegration(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test")
	tti := NewTTSIntegration(mockProvider)
	assert.NotNil(t, tti)
}

func TestNewTTSIntegration_NilProvider(t *testing.T) {
	tti := NewTTSIntegration(nil)
	assert.NotNil(t, tti)
}

func TestTTSIntegration_GenerateSpeech_Success(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test")
	tti := NewTTSIntegration(mockProvider)

	ctx := context.Background()
	audio, err := tti.GenerateSpeech(ctx, "Hello, world!")
	assert.NoError(t, err)
	assert.NotNil(t, audio)
}

func TestTTSIntegration_GenerateSpeech_NilProvider(t *testing.T) {
	tti := NewTTSIntegration(nil)

	ctx := context.Background()
	_, err := tti.GenerateSpeech(ctx, "Hello, world!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not set")
}

func TestTTSIntegration_StreamGenerate_Success(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test")
	tti := NewTTSIntegration(mockProvider)

	ctx := context.Background()
	reader, err := tti.StreamGenerate(ctx, "Hello, world!")
	assert.NoError(t, err)
	assert.NotNil(t, reader)

	// Read from reader
	data := make([]byte, 100)
	n, err := reader.Read(data)
	assert.NoError(t, err)
	assert.Greater(t, n, 0)
}

func TestTTSIntegration_StreamGenerate_NilProvider(t *testing.T) {
	tti := NewTTSIntegration(nil)

	ctx := context.Background()
	_, err := tti.StreamGenerate(ctx, "Hello, world!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not set")
}

func TestTTSIntegration_StreamGenerate_ProviderError(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test", tts.WithError(errors.New("TTS error")))
	tti := NewTTSIntegration(mockProvider)

	ctx := context.Background()
	_, err := tti.StreamGenerate(ctx, "Hello, world!")
	assert.Error(t, err)
}

func TestTTSIntegration_ConcurrentAccess(t *testing.T) {
	mockProvider := tts.NewAdvancedMockTTSProvider("test")
	tti := NewTTSIntegration(mockProvider)

	ctx := context.Background()
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = tti.GenerateSpeech(ctx, "test")
			_, _ = tti.StreamGenerate(ctx, "test")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

