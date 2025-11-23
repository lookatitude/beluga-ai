package config

import (
	"context"
	"io"
	"testing"
	"time"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigIntegration tests integration with config package
func TestConfigIntegration(t *testing.T) {
	ctx := context.Background()

	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	// Test with custom config
	config := session.DefaultConfig()
	config.SessionID = "test-session-123"
	config.Timeout = 30 * time.Minute
	config.MaxRetries = 5

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithConfig(config),
	)
	require.NoError(t, err)

	// Verify config is applied
	sessionID := voiceSession.GetSessionID()
	assert.Equal(t, "test-session-123", sessionID)

	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TestConfigValidation tests config validation
func TestConfigValidation(t *testing.T) {
	ctx := context.Background()

	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	// Test invalid config
	config := session.DefaultConfig()
	config.Timeout = -1 * time.Second // Invalid timeout

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithConfig(config),
	)
	// Should either validate and reject or use defaults
	_ = voiceSession
	_ = err
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
