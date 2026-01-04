package session

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/session/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamingSTT_StartStop(t *testing.T) {
	mockProvider := &mockSTTProvider{}
	streamingSTT := internal.NewStreamingSTT(mockProvider)

	ctx := context.Background()
	err := streamingSTT.Start(ctx)
	require.NoError(t, err)
	assert.True(t, streamingSTT.IsActive())

	err = streamingSTT.Stop()
	require.NoError(t, err)
	assert.False(t, streamingSTT.IsActive())
}

func TestStreamingTTS_StartStop(t *testing.T) {
	mockProvider := &mockTTSProvider{}
	streamingTTS := internal.NewStreamingTTS(mockProvider)

	ctx := context.Background()
	reader, err := streamingTTS.StartStream(ctx, "test text")
	require.NoError(t, err)
	assert.NotNil(t, reader)
	assert.True(t, streamingTTS.IsActive())

	streamingTTS.Stop()
	assert.False(t, streamingTTS.IsActive())
}

// TestStreamingAgent_StartStop tests streaming agent with agent instance (new API).
// The deprecated callback-based approach is no longer tested here.
func TestStreamingAgent_StartStop(t *testing.T) {
	t.Skip("Skipping deprecated test - use agent instance-based tests in internal/streaming_agent_test.go instead")
	
	// This test previously used the deprecated NewStreamingAgentWithCallback API.
	// New tests should use the agent instance-based approach.
	// See: pkg/voice/session/internal/streaming_agent_test.go for current tests.
}
