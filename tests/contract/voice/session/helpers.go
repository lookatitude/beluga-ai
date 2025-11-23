package session

import (
	"context"
	"io"
	"testing"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	voicesession "github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/require"
)

// createTestSession creates a test session with mock providers
func createTestSession(t *testing.T) voiceiface.VoiceSession {
	ctx := context.Background()

	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	sess, err := voicesession.NewVoiceSession(ctx,
		voicesession.WithSTTProvider(sttProvider),
		voicesession.WithTTSProvider(ttsProvider),
	)
	require.NoError(t, err)
	return sess
}

// Mock providers
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
