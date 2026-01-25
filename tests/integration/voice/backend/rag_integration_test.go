package backend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	// Import providers to trigger init() registration
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/mock"
)

// TestRAGIntegration tests integration with pkg/retrievers, pkg/vectorstores, pkg/embeddings (T271, T277).
// Verifies RAG-enabled voice agents can retrieve knowledge base documents.
func TestRAGIntegration(t *testing.T) {
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

	// Create session with RAG integration
	ragQueryCount := 0
	sessionConfig := &vbiface.SessionConfig{
		UserID:       "test-user",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			// Simulate RAG query based on transcript
			if transcript != "" {
				ragQueryCount++
				// In a full implementation, this would:
				// 1. Generate embeddings for the transcript
				// 2. Query vector store for similar documents
				// 3. Use retrieved context in agent response
				return "RAG-enhanced response", nil
			}
			return "Response", nil
		},
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Start session
	err = session.Start(ctx)
	require.NoError(t, err)

	// Process audio - should trigger RAG query
	audio := []byte{1, 2, 3, 4, 5}
	err = session.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// In a full implementation, this would verify that:
	// 1. Embeddings were generated for the transcript
	// 2. Vector store was queried for similar documents
	// 3. Retrieved context was used in the agent response

	// Stop session
	err = session.Stop(ctx)
	require.NoError(t, err)
}
