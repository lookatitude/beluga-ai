package s2s

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLLMForAgent is a mock LLM for agent integration tests.
type mockLLMForAgent struct {
	responses []string
	index     int
}

func (m *mockLLMForAgent) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	response := "agent response"
	if m.responses != nil && m.index < len(m.responses) {
		response = m.responses[m.index]
		m.index++
	}
	return response, nil
}

func (m *mockLLMForAgent) GetModelName() string {
	return "mock-llm"
}

func (m *mockLLMForAgent) GetProviderName() string {
	return "mock-provider"
}

func (m *mockLLMForAgent) StreamChat(ctx context.Context, messages []schema.Message, options ...interface{}) (<-chan llmsiface.AIMessageChunk, error) {
	ch := make(chan llmsiface.AIMessageChunk, 1)
	response := "agent response"
	if m.responses != nil && m.index < len(m.responses) {
		response = m.responses[m.index]
		m.index++
	}
	ch <- llmsiface.AIMessageChunk{
		Content: response,
	}
	close(ch)
	return ch, nil
}

// TestS2S_AgentIntegration_BuiltIn tests S2S with built-in reasoning mode.
func TestS2S_AgentIntegration_BuiltIn(t *testing.T) {
	ctx := context.Background()

	// Create S2S provider
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create voice session with S2S provider (built-in reasoning)
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio - should use built-in reasoning
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TestS2S_AgentIntegration_External tests S2S with external agent reasoning.
func TestS2S_AgentIntegration_External(t *testing.T) {
	ctx := context.Background()

	// Create S2S provider
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create LLM
	llm := &mockLLMForAgent{
		responses: []string{"Hello! How can I help you?"},
	}

	// Create agent
	agent, err := agents.NewBaseAgent("s2s-agent", llm, nil)
	require.NoError(t, err)

	// Create voice session with S2S provider and agent (external reasoning)
	agentConfig := &schema.AgentConfig{
		Name:            "s2s-agent",
		LLMProviderName: "mock",
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
		session.WithAgentInstance(agent.(agentsiface.StreamingAgent), agentConfig),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio - should route through agent for external reasoning
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TestS2S_AgentIntegration_Streaming tests S2S with streaming agent.
func TestS2S_AgentIntegration_Streaming(t *testing.T) {
	ctx := context.Background()

	// Create S2S provider with streaming support
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create streaming LLM
	llm := &mockLLMForAgent{
		responses: []string{"Streaming response"},
	}

	// Create streaming agent
	agent, err := agents.NewBaseAgent("s2s-streaming-agent", llm, nil,
		agents.WithStreaming(true),
		agents.WithStreamingConfig(agentsiface.StreamingConfig{
			EnableStreaming:     true,
			ChunkBufferSize:     10,
			SentenceBoundary:    true,
			InterruptOnNewInput: true,
		}),
	)
	require.NoError(t, err)

	streamingAgent, ok := agent.(agentsiface.StreamingAgent)
	require.True(t, ok)

	// Create voice session with streaming S2S and streaming agent
	agentConfig := &schema.AgentConfig{
		Name:            "s2s-streaming-agent",
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

	// Process audio
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TestS2S_AgentIntegration_ReasoningModeSwitch tests switching between reasoning modes.
func TestS2S_AgentIntegration_ReasoningModeSwitch(t *testing.T) {
	ctx := context.Background()

	// Create S2S provider
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create agent
	llm := &mockLLMForAgent{}
	agent, err := agents.NewBaseAgent("s2s-agent", llm, nil)
	require.NoError(t, err)

	// Test built-in mode
	voiceSession1, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession1)

	// Test external mode
	agentConfig := &schema.AgentConfig{
		Name:            "s2s-agent",
		LLMProviderName: "mock",
	}

	streamingAgent, ok := agent.(agentsiface.StreamingAgent)
	if !ok {
		t.Skip("Agent does not implement StreamingAgent")
		return
	}

	voiceSession2, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
		session.WithAgentInstance(streamingAgent, agentConfig),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession2)
}

// TestS2S_AgentIntegration_AgentCallback tests S2S with agent callback.
func TestS2S_AgentIntegration_AgentCallback(t *testing.T) {
	ctx := context.Background()

	// Create S2S provider
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create agent callback
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "callback response", nil
	}

	// Create voice session with S2S provider and agent callback
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
		session.WithAgentCallback(agentCallback),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TestS2S_AgentIntegration_CrossPackage tests S2S agent integration across packages.
func TestS2S_AgentIntegration_CrossPackage(t *testing.T) {
	ctx := context.Background()

	// Create S2S provider
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create LLM
	llm := &mockLLMForAgent{
		responses: []string{"Cross-package response"},
	}

	// Create agent
	agent, err := agents.NewBaseAgent("cross-package-agent", llm, nil)
	require.NoError(t, err)

	// Create voice session with S2S and agent
	agentConfig := &schema.AgentConfig{
		Name:            "cross-package-agent",
		LLMProviderName: "mock",
	}

	streamingAgent, ok := agent.(agentsiface.StreamingAgent)
	if !ok {
		t.Skip("Agent does not implement StreamingAgent")
		return
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
		session.WithAgentInstance(streamingAgent, agentConfig),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Verify cross-package integration
	assert.NotNil(t, voiceSession)

	// Start and process
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}
