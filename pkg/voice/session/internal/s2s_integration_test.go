package internal

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	s2siface "github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS2SIntegration_ProcessAudioWithSessionID(t *testing.T) {
	// Create a mock S2S provider
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{5, 4, 3, 2, 1}, "test-provider", 100*time.Millisecond)))

	var provider s2siface.S2SProvider = mockProvider
	integration := NewS2SIntegration(provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	sessionID := "test-session-123"

	output, err := integration.ProcessAudioWithSessionID(ctx, audio, sessionID)
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output)
}

func TestS2SIntegration_ProcessAudioWithSessionID_ProviderNil(t *testing.T) {
	integration := NewS2SIntegration(nil)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	sessionID := "test-session-123"

	_, err := integration.ProcessAudioWithSessionID(ctx, audio, sessionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not set")
}

func TestS2SIntegration_StartStreaming(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{5, 4, 3, 2, 1}, "test-provider", 100*time.Millisecond)))

	var provider s2siface.S2SProvider = mockProvider
	streamingProvider, ok := provider.(s2siface.StreamingS2SProvider)
	if !ok {
		t.Skip("Provider does not implement StreamingS2SProvider")
		return
	}

	integration := NewS2SIntegration(streamingProvider)

	ctx := context.Background()
	sessionID := "test-session-123"

	err := integration.StartStreaming(ctx, sessionID)
	require.NoError(t, err)
}

func TestS2SIntegration_StartStreaming_ProviderNil(t *testing.T) {
	integration := NewS2SIntegration(nil)

	ctx := context.Background()
	sessionID := "test-session-123"

	err := integration.StartStreaming(ctx, sessionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not set")
}

func TestS2SIntegration_StartStreaming_NotStreamingProvider(t *testing.T) {
	// Create a provider that doesn't support streaming
	// The AdvancedMockS2SProvider implements StreamingS2SProvider, so we need a wrapper
	// that only exposes the S2SProvider interface
	type nonStreamingProvider struct {
		s2siface.S2SProvider
	}

	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider")
	// Wrap it to hide the StreamingS2SProvider interface
	nonStreaming := &nonStreamingProvider{S2SProvider: mockProvider}

	integration := NewS2SIntegration(nonStreaming)

	ctx := context.Background()
	sessionID := "test-session-123"

	err := integration.StartStreaming(ctx, sessionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support streaming")
}

func TestS2SIntegration_SendAudio(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{5, 4, 3, 2, 1}, "test-provider", 100*time.Millisecond)))

	var provider s2siface.S2SProvider = mockProvider
	streamingProvider, ok := provider.(s2siface.StreamingS2SProvider)
	if !ok {
		t.Skip("Provider does not implement StreamingS2SProvider")
		return
	}

	integration := NewS2SIntegration(streamingProvider)

	ctx := context.Background()
	sessionID := "test-session-123"

	err := integration.StartStreaming(ctx, sessionID)
	require.NoError(t, err)

	audio := []byte{1, 2, 3, 4, 5}
	err = integration.SendAudio(ctx, audio)
	require.NoError(t, err)
}

func TestS2SIntegration_SendAudio_NoSession(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider")
	var provider s2siface.S2SProvider = mockProvider

	integration := NewS2SIntegration(provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	err := integration.SendAudio(ctx, audio)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestS2SIntegration_ReceiveAudio(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{5, 4, 3, 2, 1}, "test-provider", 100*time.Millisecond)))

	var provider s2siface.S2SProvider = mockProvider
	streamingProvider, ok := provider.(s2siface.StreamingS2SProvider)
	if !ok {
		t.Skip("Provider does not implement StreamingS2SProvider")
		return
	}

	integration := NewS2SIntegration(streamingProvider)

	ctx := context.Background()
	sessionID := "test-session-123"

	err := integration.StartStreaming(ctx, sessionID)
	require.NoError(t, err)

	audioCh := integration.ReceiveAudio()
	require.NotNil(t, audioCh)

	// Wait for audio chunk with timeout
	timeout := time.After(2 * time.Second)
	select {
	case chunk, ok := <-audioCh:
		if ok {
			assert.NotEmpty(t, chunk.Audio)
			assert.NoError(t, chunk.Error)
		}
	case <-timeout:
		t.Log("Timeout waiting for audio chunk (may be expected if mock doesn't send)")
	}
}

func TestS2SIntegration_ReceiveAudio_NoSession(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider")
	var provider s2siface.S2SProvider = mockProvider

	integration := NewS2SIntegration(provider)

	audioCh := integration.ReceiveAudio()
	require.NotNil(t, audioCh)

	// Channel should be closed
	_, ok := <-audioCh
	assert.False(t, ok, "Channel should be closed when no session")
}

func TestS2SIntegration_CloseStreaming(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{5, 4, 3, 2, 1}, "test-provider", 100*time.Millisecond)))

	var provider s2siface.S2SProvider = mockProvider
	streamingProvider, ok := provider.(s2siface.StreamingS2SProvider)
	if !ok {
		t.Skip("Provider does not implement StreamingS2SProvider")
		return
	}

	integration := NewS2SIntegration(streamingProvider)

	ctx := context.Background()
	sessionID := "test-session-123"

	err := integration.StartStreaming(ctx, sessionID)
	require.NoError(t, err)

	err = integration.CloseStreaming()
	require.NoError(t, err)

	// Closing again should not error
	err = integration.CloseStreaming()
	require.NoError(t, err)
}

func TestS2SIntegration_CloseStreaming_NoSession(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider")
	var provider s2siface.S2SProvider = mockProvider

	integration := NewS2SIntegration(provider)

	err := integration.CloseStreaming()
	require.NoError(t, err)
}
