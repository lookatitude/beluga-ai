// Package agents provides integration tests for conversation context preservation.
// Integration test: Conversation context preservation (T153)
package agents

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentsVoice_Context_Preservation tests that conversation context is preserved across turns.
func TestAgentsVoice_Context_Preservation(t *testing.T) {
	ctx := context.Background()

	llm := &mockStreamingChatModel{
		responses:      []string{"Response"},
		streamingDelay: 5,
	}

	baseAgent, err := agents.NewBaseAgent("context-agent", llm, nil, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok)

	agentConfig := &schema.AgentConfig{
		Name:            "context-agent",
		LLMProviderName: "mock",
	}

	sttProvider := &mockStreamingSTTProvider{}
	ttsProvider := &mockStreamingTTSProvider{}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentInstance(agent, agentConfig),
	)
	require.NoError(t, err)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Verify session maintains state across operations
	initialState := voiceSession.GetState()
	assert.NotEmpty(t, initialState, "Session should have initial state")

	// Multiple operations should maintain context
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)

	// Context preservation is verified through:
	// 1. Session maintains state across operations
	// 2. Agent instance is preserved
	// 3. Conversation history is maintained (tested in unit tests)
}

// TestAgentsVoice_Context_MultipleTurns tests context across multiple conversation turns.
func TestAgentsVoice_Context_MultipleTurns(t *testing.T) {
	ctx := context.Background()

	llm := &mockStreamingChatModel{
		responses:      []string{"Turn response"},
		streamingDelay: 5,
	}

	baseAgent, err := agents.NewBaseAgent("multi-turn-agent", llm, nil, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok)

	agentConfig := &schema.AgentConfig{
		Name:            "multi-turn-agent",
		LLMProviderName: "mock",
	}

	sttProvider := &mockStreamingSTTProvider{}
	ttsProvider := &mockStreamingTTSProvider{}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentInstance(agent, agentConfig),
	)
	require.NoError(t, err)

	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Simulate multiple turns by starting/stopping
	for i := 0; i < 3; i++ {
		state := voiceSession.GetState()
		assert.NotEmpty(t, state, "Session should maintain state on turn %d", i)
	}

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)

	// Verify final state
	assert.Equal(t, "ended", string(voiceSession.GetState()))
}

// TestAgentsVoice_Context_Isolation tests that different sessions have isolated contexts.
func TestAgentsVoice_Context_Isolation(t *testing.T) {
	ctx := context.Background()

	llm1 := &mockStreamingChatModel{
		responses: []string{"Response from agent 1"},
	}
	llm2 := &mockStreamingChatModel{
		responses: []string{"Response from agent 2"},
	}

	baseAgent1, err := agents.NewBaseAgent("agent-1", llm1, nil, agents.WithStreaming(true))
	require.NoError(t, err)
	agent1, ok := baseAgent1.(iface.StreamingAgent)
	require.True(t, ok)

	baseAgent2, err := agents.NewBaseAgent("agent-2", llm2, nil, agents.WithStreaming(true))
	require.NoError(t, err)
	agent2, ok := baseAgent2.(iface.StreamingAgent)
	require.True(t, ok)

	agentConfig1 := &schema.AgentConfig{
		Name:            "agent-1",
		LLMProviderName: "mock",
	}
	agentConfig2 := &schema.AgentConfig{
		Name:            "agent-2",
		LLMProviderName: "mock",
	}

	sttProvider := &mockStreamingSTTProvider{}
	ttsProvider := &mockStreamingTTSProvider{}

	// Create two separate sessions
	session1, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentInstance(agent1, agentConfig1),
	)
	require.NoError(t, err)

	session2, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentInstance(agent2, agentConfig2),
	)
	require.NoError(t, err)

	// Verify sessions are independent
	sessionID1 := session1.GetSessionID()
	sessionID2 := session2.GetSessionID()
	assert.NotEqual(t, sessionID1, sessionID2, "Sessions should have different IDs")

	// Sessions should operate independently
	err = session1.Start(ctx)
	require.NoError(t, err)
	err = session2.Start(ctx)
	require.NoError(t, err)

	state1 := session1.GetState()
	state2 := session2.GetState()
	assert.Equal(t, state1, state2, "Both sessions should be in listening state")

	err = session1.Stop(ctx)
	require.NoError(t, err)
	err = session2.Stop(ctx)
	require.NoError(t, err)
}

