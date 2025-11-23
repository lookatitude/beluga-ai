package quickstart

import (
	"context"
	"testing"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSessionLifecycle_Validation validates session lifecycle from quickstart
func TestSessionLifecycle_Validation(t *testing.T) {
	ctx := context.Background()

	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
	)
	require.NoError(t, err)

	// Initial state
	assert.Equal(t, "initial", string(voiceSession.GetState()))

	// Start -> Listening
	err = voiceSession.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, "listening", string(voiceSession.GetState()))

	// Process audio -> may transition to processing
	audio := []byte{1, 2, 3}
	err = voiceSession.ProcessAudio(ctx, audio)
	assert.NoError(t, err)

	// Say -> may transition to speaking
	handle, err := voiceSession.Say(ctx, "test")
	require.NoError(t, err)
	assert.NotNil(t, handle)

	// Stop -> Ended
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "ended", string(voiceSession.GetState()))
}

// TestStateTransitions_Validation validates state transitions
func TestStateTransitions_Validation(t *testing.T) {
	ctx := context.Background()

	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
	)
	require.NoError(t, err)

	// Test state change callback
	stateChanges := []string{}
	voiceSession.OnStateChanged(func(state voiceiface.SessionState) {
		stateChanges = append(stateChanges, string(state))
	})

	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Should have received listening state
	assert.Contains(t, stateChanges, "listening")

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)

	// Should have received ended state
	assert.Contains(t, stateChanges, "ended")
}
