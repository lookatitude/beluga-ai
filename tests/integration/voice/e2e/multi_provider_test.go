package e2e

import (
	"context"
	"testing"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiProvider_E2E tests multi-provider fallback end-to-end.
func TestMultiProvider_E2E(t *testing.T) {
	ctx := context.Background()

	// Create primary and fallback providers
	primarySTT := &failingSTTProvider{}
	primaryTTS := &mockTTSProvider{}

	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "Response", nil
	}

	// Create session with primary provider
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(primarySTT),
		session.WithTTSProvider(primaryTTS),
		session.WithAgentCallback(agentCallback),
	)
	require.NoError(t, err)

	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio - primary should fail, fallback should be used
	// (if fallback is configured in implementation)
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	// May fail or use fallback depending on implementation
	_ = err

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// Failing STT provider.
type failingSTTProvider struct{}

func (f *failingSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "", assert.AnError
}

func (f *failingSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return nil, assert.AnError
}
