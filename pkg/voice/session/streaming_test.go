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

func TestStreamingAgent_StartStop(t *testing.T) {
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	}

	streamingAgent := internal.NewStreamingAgent(agentCallback)

	ctx := context.Background()
	responseCh, err := streamingAgent.StartStreaming(ctx, "test")
	require.NoError(t, err)
	assert.NotNil(t, responseCh)
	assert.True(t, streamingAgent.IsStreaming())

	streamingAgent.StopStreaming()
	assert.False(t, streamingAgent.IsStreaming())
}
