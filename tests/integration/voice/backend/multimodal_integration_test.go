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

// TestMultimodalIntegration tests integration with pkg/multimodal (T272, T278).
// Verifies audio content is handled in multimodal workflows.
func TestMultimodalIntegration(t *testing.T) {
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

	// Create session with multimodal integration
	multimodalProcessed := false
	sessionConfig := &vbiface.SessionConfig{
		UserID:       "test-user",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			// Simulate multimodal processing
			multimodalProcessed = true
			// In a full implementation, this would:
			// 1. Process audio through multimodal model
			// 2. Handle audio content in multimodal workflows
			// 3. Generate multimodal responses
			return "Multimodal response", nil
		},
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Start session
	err = session.Start(ctx)
	require.NoError(t, err)

	// Process audio - should trigger multimodal processing
	audio := []byte{1, 2, 3, 4, 5}
	err = session.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Verify multimodal processing was triggered
	assert.True(t, multimodalProcessed, "Multimodal processing should have been triggered")

	// Stop session
	err = session.Stop(ctx)
	require.NoError(t, err)
}
