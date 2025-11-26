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

func TestNewStreamingSTT(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sstt := NewStreamingSTT(mockProvider)
	assert.NotNil(t, sstt)
	assert.False(t, sstt.IsActive())
}

func TestNewStreamingSTT_NilProvider(t *testing.T) {
	sstt := NewStreamingSTT(nil)
	assert.NotNil(t, sstt)
	assert.False(t, sstt.IsActive())
}

func TestStreamingSTT_Start_Success(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sstt := NewStreamingSTT(mockProvider)

	ctx := context.Background()
	err := sstt.Start(ctx)
	require.NoError(t, err)
	assert.True(t, sstt.IsActive())
}

func TestStreamingSTT_Start_AlreadyActive(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sstt := NewStreamingSTT(mockProvider)

	ctx := context.Background()
	err := sstt.Start(ctx)
	require.NoError(t, err)

	// Try to start again
	err = sstt.Start(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already active")
}

func TestStreamingSTT_Start_NilProvider(t *testing.T) {
	sstt := NewStreamingSTT(nil)

	ctx := context.Background()
	err := sstt.Start(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not set")
}

func TestStreamingSTT_SendAudio_Success(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sstt := NewStreamingSTT(mockProvider)

	ctx := context.Background()
	err := sstt.Start(ctx)
	require.NoError(t, err)

	audio := []byte{1, 2, 3, 4, 5}
	err = sstt.SendAudio(ctx, audio)
	require.NoError(t, err)
}

func TestStreamingSTT_SendAudio_NotActive(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sstt := NewStreamingSTT(mockProvider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	err := sstt.SendAudio(ctx, audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestStreamingSTT_ReceiveTranscript_Success(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sstt := NewStreamingSTT(mockProvider)

	ctx := context.Background()
	err := sstt.Start(ctx)
	require.NoError(t, err)

	ch := sstt.ReceiveTranscript()
	assert.NotNil(t, ch)

	// Should receive transcriptions
	select {
	case result := <-ch:
		require.NoError(t, result.Error)
		assert.NotEmpty(t, result.Text)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for transcript")
	}
}

func TestStreamingSTT_ReceiveTranscript_NotStarted(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sstt := NewStreamingSTT(mockProvider)

	ch := sstt.ReceiveTranscript()
	assert.NotNil(t, ch)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok)
}

func TestStreamingSTT_Stop_Success(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sstt := NewStreamingSTT(mockProvider)

	ctx := context.Background()
	err := sstt.Start(ctx)
	require.NoError(t, err)
	assert.True(t, sstt.IsActive())

	err = sstt.Stop()
	require.NoError(t, err)
	assert.False(t, sstt.IsActive())
}

func TestStreamingSTT_Stop_NotActive(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sstt := NewStreamingSTT(mockProvider)

	err := sstt.Stop()
	require.NoError(t, err)
	assert.False(t, sstt.IsActive())
}

func TestStreamingSTT_IsActive(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test")
	sstt := NewStreamingSTT(mockProvider)

	assert.False(t, sstt.IsActive())

	ctx := context.Background()
	err := sstt.Start(ctx)
	require.NoError(t, err)
	assert.True(t, sstt.IsActive())

	err = sstt.Stop()
	require.NoError(t, err)
	assert.False(t, sstt.IsActive())
}

func TestStreamingSTT_Start_ProviderError(t *testing.T) {
	mockProvider := stt.NewAdvancedMockSTTProvider("test", stt.WithError(errors.New("provider error")))
	sstt := NewStreamingSTT(mockProvider)

	ctx := context.Background()
	err := sstt.Start(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start streaming")
	assert.False(t, sstt.IsActive())
}
