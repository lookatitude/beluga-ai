package e2e

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInterruption_E2E tests interruption handling end-to-end.
func TestInterruption_E2E(t *testing.T) {
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

	// Start saying something
	handle, err := voiceSession.Say(ctx, "This is a long response that should be interruptible")
	require.NoError(t, err)
	assert.NotNil(t, handle)

	// Simulate user interruption by processing audio
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Cancel the current say operation (simulating interruption)
	err = handle.Cancel()
	require.NoError(t, err)

	// Should be able to process new audio after interruption
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}
