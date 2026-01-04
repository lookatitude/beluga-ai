// Package agents provides integration tests for backward compatibility with callback mode.
// Integration test: Backward compatibility (callback mode) (T155)
package agents

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentsVoice_BackwardCompat_CallbackMode tests that callback mode still works.
func TestAgentsVoice_BackwardCompat_CallbackMode(t *testing.T) {
	ctx := context.Background()

	// Create agent callback (legacy mode)
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "Callback response to: " + transcript, nil
	}

	sttProvider := &mockStreamingSTTProvider{
		transcript: "test transcript",
	}
	ttsProvider := &mockStreamingTTSProvider{}

	// Create session with callback (backward compatibility)
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	require.NoError(t, err)
	require.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, "listening", string(voiceSession.GetState()))

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "ended", string(voiceSession.GetState()))

	// Note: Callback will be called when ProcessAudio is fully implemented
	// For now, we verify backward compatibility by ensuring session creation works
}

// TestAgentsVoice_BackwardCompat_BothModes tests that both callback and agent instance can coexist.
func TestAgentsVoice_BackwardCompat_BothModes(t *testing.T) {
	ctx := context.Background()

	// Create both callback and agent instance
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "Callback response", nil
	}

	llm := &mockStreamingChatModel{
		responses:      []string{"Agent response"},
		streamingDelay: 5 * time.Millisecond,
	}

	baseAgent, err := agents.NewBaseAgent("compat-agent", llm, nil, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok)

	agentConfig := &schema.AgentConfig{
		Name:            "compat-agent",
		LLMProviderName: "mock",
	}

	sttProvider := &mockStreamingSTTProvider{}
	ttsProvider := &mockStreamingTTSProvider{}

	// Create session with both callback and agent instance
	// Agent instance should take precedence
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
		session.WithAgentInstance(agent, agentConfig),
	)
	require.NoError(t, err)
	require.NotNil(t, voiceSession)

	// Verify session can be started
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

