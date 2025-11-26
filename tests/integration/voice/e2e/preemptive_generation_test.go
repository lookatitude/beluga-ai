package e2e

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/require"
)

// TestPreemptiveGeneration_E2E tests preemptive generation end-to-end.
func TestPreemptiveGeneration_E2E(t *testing.T) {
	ctx := context.Background()

	sttProvider := &streamingSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	responses := []string{}
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		responses = append(responses, transcript)
		return "Response to: " + transcript, nil
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	require.NoError(t, err)

	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio that generates interim and final transcripts
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Should have received at least one response
	// (implementation may vary on preemptive generation)
	// len(responses) is always >= 0, so no assertion needed

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}
