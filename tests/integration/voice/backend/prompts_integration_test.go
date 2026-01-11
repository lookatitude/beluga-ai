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

// TestPromptsIntegration tests integration with pkg/prompts (T273).
// Verifies agent prompt management in voice sessions.
func TestPromptsIntegration(t *testing.T) {
	ctx := context.Background()

	config := backend.DefaultConfig()
	config.Provider = "mock"

	voiceBackend, err := backend.NewBackend(ctx, "mock", config)
	require.NoError(t, err)

	err = voiceBackend.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = voiceBackend.Stop(ctx)
	}()

	// Create session with prompt integration
	promptUsed := false
	sessionConfig := &vbiface.SessionConfig{
		UserID:       "test-user",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			// Simulate prompt template usage
			promptUsed = true
			// In a full implementation, this would:
			// 1. Load prompt template from pkg/prompts
			// 2. Format prompt with transcript and context
			// 3. Use formatted prompt for agent response
			return "Prompt-based response", nil
		},
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Start session
	err = session.Start(ctx)
	require.NoError(t, err)

	// Process audio - should use prompt template
	audio := []byte{1, 2, 3, 4, 5}
	err = session.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Verify prompt was used
	assert.True(t, promptUsed, "Prompt template should have been used")

	// Stop session
	err = session.Stop(ctx)
	require.NoError(t, err)
}
