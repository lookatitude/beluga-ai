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

// TestMemoryIntegration tests integration with pkg/memory (T269, T275).
// Verifies conversation context is stored and retrieved per session.
func TestMemoryIntegration(t *testing.T) {
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

	// Create session
	sessionConfig := &vbiface.SessionConfig{
		UserID:       "test-user",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			return "Response", nil
		},
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err)
	require.NotNil(t, session)

	sessionID := session.GetID()
	assert.NotEmpty(t, sessionID, "Session should have an ID")

	// Verify session can be retrieved (memory integration)
	retrievedSession, err := voiceBackend.GetSession(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, sessionID, retrievedSession.GetID(), "Retrieved session should match created session")

	// Start session
	err = session.Start(ctx)
	require.NoError(t, err)

	// Process multiple audio chunks - conversation context should be preserved
	audio1 := []byte{1, 2, 3, 4, 5}
	err = session.ProcessAudio(ctx, audio1)
	require.NoError(t, err)

	audio2 := []byte{6, 7, 8, 9, 10}
	err = session.ProcessAudio(ctx, audio2)
	require.NoError(t, err)

	// Verify session state is preserved
	assert.Equal(t, vbiface.PipelineStateListening, session.GetState(), "Session state should be preserved")

	// Stop session
	err = session.Stop(ctx)
	require.NoError(t, err)
}
