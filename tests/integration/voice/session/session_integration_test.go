package session

import (
	"context"
	"io"
	"testing"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock providers for testing.
type mockSTTProvider struct{}

func (m *mockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "", nil
}

func (m *mockSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return nil, nil
}

type mockTTSProvider struct{}

func (m *mockTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	return nil, nil
}

func (m *mockTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return nil, nil
}

func TestSession_Integration(t *testing.T) {
	// Integration test for Session creation and basic operations
	// This test uses mock providers to avoid requiring real connections

	t.Run("session creation", func(t *testing.T) {
		ctx := context.Background()

		// Create mock providers
		sttProvider := &mockSTTProvider{}
		ttsProvider := &mockTTSProvider{}

		agentCallback := func(ctx context.Context, transcript string) (string, error) {
			return "response", nil
		}

		voiceSession, err := session.NewVoiceSession(ctx,
			session.WithSTTProvider(sttProvider),
			session.WithTTSProvider(ttsProvider),
			session.WithAgentCallback(agentCallback),
		)
		require.NoError(t, err)
		assert.NotNil(t, voiceSession)
	})

	t.Run("session lifecycle", func(t *testing.T) {
		ctx := context.Background()

		sttProvider := &mockSTTProvider{}
		ttsProvider := &mockTTSProvider{}

		voiceSession, err := session.NewVoiceSession(ctx,
			session.WithSTTProvider(sttProvider),
			session.WithTTSProvider(ttsProvider),
		)
		require.NoError(t, err)

		// Start session
		err = voiceSession.Start(ctx)
		require.NoError(t, err)
		assert.Equal(t, "listening", string(voiceSession.GetState()))

		// Stop session
		err = voiceSession.Stop(ctx)
		require.NoError(t, err)
		assert.Equal(t, "ended", string(voiceSession.GetState()))
	})

	t.Run("session process audio", func(t *testing.T) {
		ctx := context.Background()

		sttProvider := &mockSTTProvider{}
		ttsProvider := &mockTTSProvider{}

		voiceSession, err := session.NewVoiceSession(ctx,
			session.WithSTTProvider(sttProvider),
			session.WithTTSProvider(ttsProvider),
		)
		require.NoError(t, err)

		err = voiceSession.Start(ctx)
		require.NoError(t, err)

		audio := []byte{1, 2, 3, 4, 5}
		err = voiceSession.ProcessAudio(ctx, audio)
		require.NoError(t, err)

		voiceSession.Stop(ctx)
	})
}

func TestSession_ErrorHandling(t *testing.T) {
	t.Run("start already active session", func(t *testing.T) {
		ctx := context.Background()

		mockSession := session.NewAdvancedMockSession("test",
			session.WithActive(true),
		)

		err := mockSession.Start(ctx)
		require.Error(t, err)
	})

	t.Run("stop inactive session", func(t *testing.T) {
		ctx := context.Background()

		mockSession := session.NewAdvancedMockSession("test",
			session.WithActive(false),
		)

		err := mockSession.Stop(ctx)
		require.Error(t, err)
	})
}

func TestSession_StateTransitions(t *testing.T) {
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

	// Start -> listening
	err = voiceSession.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, "listening", string(voiceSession.GetState()))

	// Stop -> ended
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "ended", string(voiceSession.GetState()))
}
