package internal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSTTIntegration(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sti := NewSTTIntegration(mockProvider)
	assert.NotNil(t, sti)
}

func TestNewSTTIntegration_NilProvider(t *testing.T) {
	sti := NewSTTIntegration(nil)
	assert.NotNil(t, sti)
}

func TestSTTIntegration_Transcribe_Success(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test", stt.WithTranscriptions("Hello world"))
	sti := NewSTTIntegration(mockProvider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	text, err := sti.Transcribe(ctx, audio)
	assert.NoError(t, err)
	assert.Equal(t, "Hello world", text)
}

func TestSTTIntegration_Transcribe_NilProvider(t *testing.T) {
	sti := NewSTTIntegration(nil)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	_, err := sti.Transcribe(ctx, audio)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not set")
}

func TestSTTIntegration_StartStreaming_Success(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sti := NewSTTIntegration(mockProvider)

	ctx := context.Background()
	err := sti.StartStreaming(ctx)
	assert.NoError(t, err)
}

func TestSTTIntegration_StartStreaming_NilProvider(t *testing.T) {
	sti := NewSTTIntegration(nil)

	ctx := context.Background()
	err := sti.StartStreaming(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not set")
}

func TestSTTIntegration_StartStreaming_ProviderError(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test", stt.WithError(errors.New("provider error")))
	sti := NewSTTIntegration(mockProvider)

	ctx := context.Background()
	err := sti.StartStreaming(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start")
}

func TestSTTIntegration_SendAudio_Success(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sti := NewSTTIntegration(mockProvider)

	ctx := context.Background()
	err := sti.StartStreaming(ctx)
	require.NoError(t, err)

	audio := []byte{1, 2, 3, 4, 5}
	err = sti.SendAudio(ctx, audio)
	assert.NoError(t, err)
}

func TestSTTIntegration_SendAudio_NotStarted(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sti := NewSTTIntegration(mockProvider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	err := sti.SendAudio(ctx, audio)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestSTTIntegration_ReceiveTranscript_Success(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test", stt.WithTranscriptions("Hello"))
	sti := NewSTTIntegration(mockProvider)

	ctx := context.Background()
	err := sti.StartStreaming(ctx)
	require.NoError(t, err)

	ch := sti.ReceiveTranscript()
	assert.NotNil(t, ch)

	// Should receive transcriptions
	select {
	case result := <-ch:
		assert.NoError(t, result.Error)
		assert.NotEmpty(t, result.Text)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for transcript")
	}
}

func TestSTTIntegration_ReceiveTranscript_NotStarted(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sti := NewSTTIntegration(mockProvider)

	ch := sti.ReceiveTranscript()
	assert.NotNil(t, ch)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok)
}

func TestSTTIntegration_CloseStreaming_Success(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sti := NewSTTIntegration(mockProvider)

	ctx := context.Background()
	err := sti.StartStreaming(ctx)
	require.NoError(t, err)

	err = sti.CloseStreaming()
	assert.NoError(t, err)
}

func TestSTTIntegration_CloseStreaming_NotStarted(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sti := NewSTTIntegration(mockProvider)

	err := sti.CloseStreaming()
	assert.NoError(t, err)
}

