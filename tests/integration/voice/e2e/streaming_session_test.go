package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStreamingSession_E2E tests streaming voice session functionality.
func TestStreamingSession_E2E(t *testing.T) {
	ctx := context.Background()

	// Create streaming-capable providers
	sttProvider := &streamingSTTProvider{} // Defined in helpers.go
	ttsProvider := &mockTTSProvider{}      // Defined in helpers.go

	// Create agent callback that returns streaming responses
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "Streaming response", nil
	}

	// Create session
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	require.NoError(t, err)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process streaming audio
	audioChunks := [][]byte{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}

	for _, chunk := range audioChunks {
		err = voiceSession.ProcessAudio(ctx, chunk)
		require.NoError(t, err)
	}

	// Test streaming TTS
	handle, err := voiceSession.Say(ctx, "Streaming test")
	require.NoError(t, err)
	assert.NotNil(t, handle)

	// Wait for playout with timeout
	playoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	_ = handle.WaitForPlayout(playoutCtx)

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}
