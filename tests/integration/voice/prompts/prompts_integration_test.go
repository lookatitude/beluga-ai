package prompts

import (
	"context"
	"io"
	"testing"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/require"
)

// TestPromptsIntegration tests integration with prompts package
// This is a placeholder - actual integration would depend on prompts package API.
func TestPromptsIntegration(t *testing.T) {
	ctx := context.Background()

	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	// Agent callback simulates prompt integration
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		// In real implementation, this would use prompts package to:
		// 1. Format system prompts
		// 2. Format user messages
		// 3. Apply prompt templates
		return "Prompted response to: " + transcript, nil
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	require.NoError(t, err)

	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio - should use prompts for agent interaction
	audio := []byte{1, 2, 3}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TODO: In real implementation, this would test actual prompts package integration:
// - Prompt template formatting
// - System prompt configuration
// - User message formatting
// - Prompt variable substitution

// Mock providers.
type mockSTTProvider struct{}

func (m *mockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "test transcript", nil
}

func (m *mockSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return nil, nil
}

type mockTTSProvider struct{}

func (m *mockTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	return []byte{1, 2, 3}, nil
}

func (m *mockTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return nil, nil
}
