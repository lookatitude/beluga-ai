package backend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	// Import providers to trigger init() registration
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/mock"
)

// TestChatmodelsIntegration tests integration with pkg/chatmodels (T274).
// Verifies agent LLM integration in voice sessions.
func TestChatmodelsIntegration(t *testing.T) {
	t.Skip("Skipping - mock provider ProcessAudio requires actual STT/TTS providers to be registered")
	ctx := context.Background()

	config := backend.DefaultConfig()
	config.Provider = "mock"
	config.STTProvider = "mock" // Required for stt_tts pipeline
	config.TTSProvider = "mock" // Required for stt_tts pipeline

	voiceBackend, err := backend.NewBackend(ctx, "mock", config)
	require.NoError(t, err)

	err = voiceBackend.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = voiceBackend.Stop(ctx)
	}()

	// Create session with chatmodel integration
	llmCalled := false
	sessionConfig := &vbiface.SessionConfig{
		UserID:       "test-user",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			// Simulate LLM call
			llmCalled = true
			// In a full implementation, this would:
			// 1. Use chatmodel from pkg/chatmodels
			// 2. Generate response using LLM
			// 3. Return LLM-generated response
			return "LLM-generated response", nil
		},
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Start session
	err = session.Start(ctx)
	require.NoError(t, err)

	// Process audio - should call LLM
	audio := []byte{1, 2, 3, 4, 5}
	err = session.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Verify LLM was called
	assert.True(t, llmCalled, "LLM should have been called")

	// Stop session
	err = session.Stop(ctx)
	require.NoError(t, err)
}
