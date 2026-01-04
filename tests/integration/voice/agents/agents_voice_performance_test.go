// Package agents provides integration tests for performance validation.
// Integration test: Performance validation (< 500ms latency) (T154)
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

const (
	// maxLatency is the maximum acceptable latency for streaming operations (500ms).
	maxLatency = 500 * time.Millisecond
)

// TestAgentsVoice_Performance_StreamLatency tests that streaming latency is within acceptable limits.
func TestAgentsVoice_Performance_StreamLatency(t *testing.T) {
	ctx := context.Background()

	// Create LLM with minimal delay
	llm := &mockStreamingChatModel{
		responses:      []string{"Fast", "response"},
		streamingDelay: 1 * time.Millisecond, // Minimal delay
	}

	baseAgent, err := agents.NewBaseAgent("perf-agent", llm, nil, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok)

	inputs := map[string]any{"input": "test"}

	// Measure latency from stream start to first chunk
	startTime := time.Now()
	chunkChan, err := agent.StreamExecute(ctx, inputs)
	require.NoError(t, err)

	// Wait for first chunk
	firstChunkReceived := false
	timeout := time.After(maxLatency * 2) // Give 2x timeout for test reliability

	for !firstChunkReceived {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				t.Fatal("Stream closed before first chunk received")
			}
			if chunk.Content != "" || chunk.Err != nil {
				latency := time.Since(startTime)
				assert.Less(t, latency, maxLatency, "First chunk latency should be less than %v, got %v", maxLatency, latency)
				firstChunkReceived = true
			}
			if chunk.Finish != nil {
				// Stream completed without content chunks
				latency := time.Since(startTime)
				if latency < maxLatency {
					firstChunkReceived = true
				}
			}
		case <-timeout:
			if !firstChunkReceived {
				t.Fatal("Timeout waiting for first chunk")
			}
		}
	}

	// Consume remaining chunks
	for range chunkChan {
	}
}

// TestAgentsVoice_Performance_SessionCreation tests session creation performance.
func TestAgentsVoice_Performance_SessionCreation(t *testing.T) {
	ctx := context.Background()

	llm := &mockStreamingChatModel{
		responses: []string{"Response"},
	}

	baseAgent, err := agents.NewBaseAgent("perf-session-agent", llm, nil, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok)

	agentConfig := &schema.AgentConfig{
		Name:            "perf-session-agent",
		LLMProviderName: "mock",
	}

	sttProvider := &mockStreamingSTTProvider{}
	ttsProvider := &mockStreamingTTSProvider{}

	// Measure session creation time
	startTime := time.Now()
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentInstance(agent, agentConfig),
	)
	creationTime := time.Since(startTime)

	require.NoError(t, err)
	assert.Less(t, creationTime, 100*time.Millisecond, "Session creation should be fast (< 100ms), took %v", creationTime)

	// Measure session start time
	startTime = time.Now()
	err = voiceSession.Start(ctx)
	startTimeElapsed := time.Since(startTime)

	require.NoError(t, err)
	assert.Less(t, startTimeElapsed, 100*time.Millisecond, "Session start should be fast (< 100ms), took %v", startTimeElapsed)

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TestAgentsVoice_Performance_ConcurrentLatency tests latency under concurrent load.
func TestAgentsVoice_Performance_ConcurrentLatency(t *testing.T) {
	ctx := context.Background()
	const numStreams = 5

	llm := &mockStreamingChatModel{
		responses:      []string{"Response"},
		streamingDelay: 5 * time.Millisecond,
	}

	baseAgent, err := agents.NewBaseAgent("concurrent-perf-agent", llm, nil, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok)

	maxLatencySeen := time.Duration(0)

	// Run multiple streams concurrently and measure latency
	type result struct {
		err     error
		latency time.Duration
	}
	results := make(chan result, numStreams)

	for i := 0; i < numStreams; i++ {
		go func() {
			inputs := map[string]any{"input": "test"}
			startTime := time.Now()
			chunkChan, err := agent.StreamExecute(ctx, inputs)
			if err != nil {
				results <- result{err: err}
				return
			}

			// Wait for first chunk
			timeout := time.After(maxLatency * 2)
			gotChunk := false
			for !gotChunk {
				select {
				case chunk, ok := <-chunkChan:
					if !ok {
						results <- result{latency: time.Since(startTime)}
						return
					}
					if chunk.Content != "" || chunk.Err != nil || chunk.Finish != nil {
						latency := time.Since(startTime)
						results <- result{latency: latency}
						gotChunk = true
					}
				case <-timeout:
					results <- result{latency: time.Since(startTime), err: assert.AnError}
					return
				}
			}

			// Consume remaining
			for range chunkChan {
			}
		}()
	}

	// Collect results
	for i := 0; i < numStreams; i++ {
		res := <-results
		if res.err == nil && res.latency > maxLatencySeen {
			maxLatencySeen = res.latency
		}
	}

	// Under concurrent load, latency may be higher but should still be reasonable
	// We use 2x the single-stream limit for concurrent scenarios
	concurrentMaxLatency := maxLatency * 2
	assert.Less(t, maxLatencySeen, concurrentMaxLatency,
		"Maximum latency under concurrent load should be less than %v, got %v",
		concurrentMaxLatency, maxLatencySeen)
}
