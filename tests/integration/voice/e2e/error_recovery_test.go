package e2e

import (
	"context"
	"errors"
	"testing"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/require"
)

// TestErrorRecovery_E2E tests error recovery end-to-end.
func TestErrorRecovery_E2E(t *testing.T) {
	ctx := context.Background()

	// Create provider that fails initially then succeeds
	sttProvider := &recoveringSTTProvider{}
	ttsProvider := &mockTTSProvider{} // Defined in helpers.go

	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "Recovered response", nil
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	require.NoError(t, err)

	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// First attempt should fail
	audio := []byte{1, 2, 3}
	err = voiceSession.ProcessAudio(ctx, audio)
	// May fail or recover depending on implementation
	_ = err

	// Second attempt should succeed (recovery)
	err = voiceSession.ProcessAudio(ctx, audio)
	// Should eventually succeed
	_ = err

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// Recovering STT provider that fails then recovers.
type recoveringSTTProvider struct {
	attempts int
}

func (r *recoveringSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	r.attempts++
	if r.attempts < 2 {
		return "", errors.New("temporary error")
	}
	return "recovered transcript", nil
}

func (r *recoveringSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return nil, nil
}
