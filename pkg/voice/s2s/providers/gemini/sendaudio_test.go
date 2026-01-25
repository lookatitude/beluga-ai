package gemini

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/providers/internal/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiNativeProvider_SendAudio_Basic(t *testing.T) {
	t.Skip("Skipping - Gemini streaming requires websocket client mock or actual API credentials")
	config := &s2s.Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}
	provider, err := NewGeminiNativeProvider(config, WithHTTPClient(mock.NewMockHTTPClient()))
	require.NoError(t, err)

	streamingProvider, ok := provider.(iface.StreamingS2SProvider)
	require.True(t, ok, "Provider should implement StreamingS2SProvider")

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	ctx := context.Background()
	session, err := streamingProvider.StartStreaming(ctx, convCtx)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer func() { _ = session.Close() }()

	// Test sending audio
	audioData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	err = session.SendAudio(ctx, audioData)
	require.NoError(t, err, "SendAudio should succeed")
}

func TestGeminiNativeProvider_SendAudio_MultipleChunks(t *testing.T) {
	t.Skip("Skipping - Gemini streaming requires websocket client mock or actual API credentials")
	config := &s2s.Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}
	provider, err := NewGeminiNativeProvider(config, WithHTTPClient(mock.NewMockHTTPClient()))
	require.NoError(t, err)

	streamingProvider, ok := provider.(iface.StreamingS2SProvider)
	require.True(t, ok)

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	ctx := context.Background()
	session, err := streamingProvider.StartStreaming(ctx, convCtx)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer func() { _ = session.Close() }()

	// Send multiple audio chunks
	chunks := [][]byte{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
		{9, 10, 11, 12},
	}

	for i, chunk := range chunks {
		err = session.SendAudio(ctx, chunk)
		require.NoError(t, err, "SendAudio should succeed for chunk %d", i)
		// Small delay to allow stream restart
		time.Sleep(50 * time.Millisecond)
	}
}

func TestGeminiNativeProvider_SendAudio_ContextCancellation(t *testing.T) {
	t.Skip("Skipping - Gemini streaming requires websocket client mock or actual API credentials")
	config := &s2s.Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}
	provider, err := NewGeminiNativeProvider(config, WithHTTPClient(mock.NewMockHTTPClient()))
	require.NoError(t, err)

	streamingProvider, ok := provider.(iface.StreamingS2SProvider)
	require.True(t, ok)

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	ctx := context.Background()
	session, err := streamingProvider.StartStreaming(ctx, convCtx)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer func() { _ = session.Close() }()

	// Test with canceled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = session.SendAudio(cancelledCtx, []byte{1, 2, 3})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "canceled")
}

func TestGeminiNativeProvider_SendAudio_ClosedSession(t *testing.T) {
	t.Skip("Skipping - Gemini streaming requires websocket client mock or actual API credentials")
	config := &s2s.Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}
	provider, err := NewGeminiNativeProvider(config, WithHTTPClient(mock.NewMockHTTPClient()))
	require.NoError(t, err)

	streamingProvider, ok := provider.(iface.StreamingS2SProvider)
	require.True(t, ok)

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	ctx := context.Background()
	session, err := streamingProvider.StartStreaming(ctx, convCtx)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Close session
	err = session.Close()
	require.NoError(t, err)

	// Try to send audio after closing
	err = session.SendAudio(ctx, []byte{1, 2, 3})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestGeminiNativeProvider_SendAudio_Accumulation(t *testing.T) {
	t.Skip("Skipping - Gemini streaming requires websocket client mock or actual API credentials")
	config := &s2s.Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}
	mockClient := mock.NewMockHTTPClient()
	provider, err := NewGeminiNativeProvider(config, WithHTTPClient(mockClient))
	require.NoError(t, err)

	streamingProvider, ok := provider.(iface.StreamingS2SProvider)
	require.True(t, ok)

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	ctx := context.Background()
	session, err := streamingProvider.StartStreaming(ctx, convCtx)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer func() { _ = session.Close() }()

	// Send multiple small chunks - they should be accumulated
	chunk1 := []byte{1, 2, 3}
	chunk2 := []byte{4, 5, 6}
	chunk3 := []byte{7, 8, 9}

	err = session.SendAudio(ctx, chunk1)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	err = session.SendAudio(ctx, chunk2)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	err = session.SendAudio(ctx, chunk3)
	require.NoError(t, err)
}
