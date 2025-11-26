package e2e

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/require"
)

// TestLongUtterance_E2E tests long utterance handling end-to-end.
func TestLongUtterance_E2E(t *testing.T) {
	ctx := context.Background()

	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "Response", nil
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	require.NoError(t, err)

	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process large audio chunk (simulating long utterance)
	largeAudio := make([]byte, 1024*1024) // 1MB
	for i := range largeAudio {
		largeAudio[i] = byte(i % 256)
	}

	err = voiceSession.ProcessAudio(ctx, largeAudio)
	// Should handle large chunks (may chunk internally)
	require.NoError(t, err)

	// Process multiple large chunks
	for i := 0; i < 10; i++ {
		chunk := make([]byte, 100*1024) // 100KB chunks
		err = voiceSession.ProcessAudio(ctx, chunk)
		require.NoError(t, err)
	}

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}
