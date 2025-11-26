package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSimpleSession_E2E tests a simple end-to-end voice session flow.
func TestSimpleSession_E2E(t *testing.T) {
	ctx := context.Background()

	// Create mock providers
	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	// Create agent callback
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "Hello! How can I help you?", nil
	}

	// Create session
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, "listening", string(voiceSession.GetState()))

	// Process audio
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Say something
	handle, err := voiceSession.Say(ctx, "Hello, user!")
	require.NoError(t, err)
	assert.NotNil(t, handle)

	// Wait for playout (with timeout)
	playoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	_ = handle.WaitForPlayout(playoutCtx)

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "ended", string(voiceSession.GetState()))
}
