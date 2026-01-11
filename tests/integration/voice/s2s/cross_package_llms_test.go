package s2s

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestS2S_LLMsIntegration tests S2S integration with LLM package.
// This validates that S2S providers can work with LLM-based agents.
func TestS2S_LLMsIntegration(t *testing.T) {
	ctx := context.Background()

	// Create S2S provider
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create mock streaming LLM
	llm := &mockStreamingLLM{}

	// Create streaming agent with LLM
	agent, err := agents.NewBaseAgent("s2s-llm-agent", llm, nil,
		agents.WithStreaming(true),
		agents.WithStreamingConfig(iface.StreamingConfig{
			EnableStreaming: true,
		}),
	)
	require.NoError(t, err)

	streamingAgent, ok := agent.(iface.StreamingAgent)
	require.True(t, ok, "Agent must implement StreamingAgent")

	// Create voice session with S2S provider and LLM-based agent
	agentConfig := &schema.AgentConfig{
		Name:            "s2s-llm-agent",
		LLMProviderName: "mock",
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
		session.WithAgentInstance(streamingAgent, agentConfig),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio - should route through S2S and then LLM agent
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TestS2S_LLMsStreaming tests S2S with streaming LLM responses.
func TestS2S_LLMsStreaming(t *testing.T) {
	ctx := context.Background()

	// Create S2S provider
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create streaming LLM that returns chunks
	llm := &mockStreamingLLM{}

	// Create streaming agent
	agent, err := agents.NewBaseAgent("s2s-streaming-agent", llm, nil,
		agents.WithStreaming(true),
		agents.WithStreamingConfig(iface.StreamingConfig{
			EnableStreaming:     true,
			ChunkBufferSize:      10,
			SentenceBoundary:    true,
			InterruptOnNewInput: true,
		}),
	)
	require.NoError(t, err)

	streamingAgent, ok := agent.(iface.StreamingAgent)
	require.True(t, ok)

	// Create voice session
	agentConfig := &schema.AgentConfig{
		Name:            "s2s-streaming-agent",
		LLMProviderName: "mock",
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
		session.WithAgentInstance(streamingAgent, agentConfig),
	)
	require.NoError(t, err)

	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio with streaming
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// Mock streaming LLM for testing
type mockStreamingLLM struct{}

func (m *mockStreamingLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return "Mock LLM response", nil
}

func (m *mockStreamingLLM) GetModelName() string {
	return "mock-model"
}

func (m *mockStreamingLLM) GetProviderName() string {
	return "mock-provider"
}

func (m *mockStreamingLLM) StreamChat(ctx context.Context, messages []schema.Message, options ...interface{}) (<-chan llmsiface.AIMessageChunk, error) {
	ch := make(chan llmsiface.AIMessageChunk, 10)
	go func() {
		defer close(ch)
		chunks := []string{"Hello", "!", "How", "can", "I", "help", "?"}
		for _, content := range chunks {
			select {
			case <-ctx.Done():
				return
			case ch <- llmsiface.AIMessageChunk{Content: content}:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
	return ch, nil
}
