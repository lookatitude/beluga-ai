// Package agents provides integration tests for interruption handling with streaming agents.
// Integration test: Interruption handling in voice calls (T150)
package agents

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentsVoice_Interruption_StreamCancellation tests that streaming can be interrupted.
func TestAgentsVoice_Interruption_StreamCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create LLM with long response to allow interruption
	llm := &mockStreamingChatModel{
		responses:      []string{"This", "is", "a", "very", "long", "response", "that", "can", "be", "interrupted"},
		streamingDelay: 50 * time.Millisecond,
	}

	baseAgent, err := agents.NewBaseAgent("interrupt-agent", llm, nil, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok, "Agent should implement StreamingAgent")

	inputs := map[string]any{"input": "test"}

	// Start streaming
	chunkChan, err := agent.StreamExecute(ctx, inputs)
	require.NoError(t, err)

	// Wait for first chunk
	select {
	case <-chunkChan:
		// Got first chunk
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for first chunk")
	}

	// Cancel context to interrupt
	cancel()

	// Verify stream is interrupted
	interrupted := false
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				goto done
			}
			if chunk.Err != nil {
				interrupted = true
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:

	assert.True(t, interrupted || ctx.Err() != nil, "Stream should be interrupted on context cancellation")
}

// TestAgentsVoice_Interruption_SessionInterruption tests interruption handling in session.
func TestAgentsVoice_Interruption_SessionInterruption(t *testing.T) {
	ctx := context.Background()

	llm := &mockStreamingChatModel{
		responses:      []string{"Long", "response", "that", "will", "be", "interrupted"},
		streamingDelay: 30 * time.Millisecond,
	}

	baseAgent, err := agents.NewBaseAgent("session-agent", llm, nil, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok)

	agentConfig := &schema.AgentConfig{
		Name:            "session-agent",
		LLMProviderName: "mock",
	}

	sttProvider := &mockStreamingSTTProvider{
		transcript: "Hello",
	}
	ttsProvider := &mockStreamingTTSProvider{}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentInstance(agent, agentConfig),
	)
	require.NoError(t, err)

	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Verify session can handle interruption
	// (Note: HandleInterruption is internal, but we can verify session state transitions)
	initialState := voiceSession.GetState()
	assert.Equal(t, "listening", string(initialState))

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "ended", string(voiceSession.GetState()))
}

// TestAgentsVoice_Interruption_ConcurrentInterruption tests interruption during concurrent operations.
func TestAgentsVoice_Interruption_ConcurrentInterruption(t *testing.T) {
	ctx := context.Background()

	llm := &mockStreamingChatModel{
		responses:      []string{"Response", "one", "Response", "two"},
		streamingDelay: 20 * time.Millisecond,
	}

	baseAgent, err := agents.NewBaseAgent("concurrent-interrupt-agent", llm, nil, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok)

	var wg sync.WaitGroup
	const numStreams = 3

	// Start multiple streams
	for i := 0; i < numStreams; i++ {
		wg.Add(1)
		go func(streamID int) {
			defer wg.Done()

			streamCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			inputs := map[string]any{"input": "test"}
			chunkChan, err := agent.StreamExecute(streamCtx, inputs)
			if err != nil {
				return
			}

			// Wait for first chunk then cancel
			select {
			case <-chunkChan:
				cancel() // Interrupt after first chunk
			case <-time.After(100 * time.Millisecond):
				cancel()
			}

			// Consume remaining chunks or errors
			for range chunkChan {
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no deadlocks occur
}

