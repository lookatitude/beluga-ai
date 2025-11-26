package quickstart

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQuickstart_Validation validates the quickstart example.
func TestQuickstart_Validation(t *testing.T) {
	ctx := context.Background()

	// This test validates the basic quickstart flow from quickstart.md
	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "Hello! How can I help you?", nil
	}

	// Create session (quickstart step 1)
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Start session (quickstart step 2)
	err = voiceSession.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, "listening", string(voiceSession.GetState()))

	// Process audio (quickstart step 3)
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Say something (quickstart step 4)
	handle, err := voiceSession.Say(ctx, "Hello, user!")
	require.NoError(t, err)
	assert.NotNil(t, handle)

	// Stop session (quickstart step 5)
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "ended", string(voiceSession.GetState()))
}
