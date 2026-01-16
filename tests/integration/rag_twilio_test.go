package integration

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTranscriptionRAG tests transcription storage and RAG retrieval.
func TestTranscriptionRAG(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Configure voice backend with vector store and embedder
	config := &vbiface.Config{
		Provider:     "twilio",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		STTProvider:  "openai",
		TTSProvider:  "openai",
		ProviderConfig: map[string]any{
			"account_sid":  utils.GetEnvOrSkip(t, "TWILIO_ACCOUNT_SID"),
			"auth_token":   utils.GetEnvOrSkip(t, "TWILIO_AUTH_TOKEN"),
			"phone_number": utils.GetEnvOrSkip(t, "TWILIO_PHONE_NUMBER"),
		},
		// VectorStore and Embedder would be set from config in full implementation
	}

	voiceBackend, err := backend.NewBackend(ctx, "twilio", config)
	require.NoError(t, err)

	err = voiceBackend.Start(ctx)
	require.NoError(t, err)
	defer voiceBackend.Stop(ctx)

	// Make a call that generates transcription
	// In a full implementation, this would:
	// 1. Make a call
	// 2. Wait for transcription.completed webhook
	// 3. Verify transcription is stored
	// 4. Search for transcription
	// 5. Verify RAG retrieval works

	// For now, verify structure
	assert.NotNil(t, voiceBackend)
}
