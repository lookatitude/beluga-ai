// Package agents provides integration tests for concurrent streaming agents with voice sessions.
// Integration test: Multiple concurrent voice calls with different agents (T149)
package agents

import (
	"context"
	"fmt"
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

// TestAgentsVoice_Concurrent_MultipleSessions tests multiple concurrent voice calls with different agents.
func TestAgentsVoice_Concurrent_MultipleSessions(t *testing.T) {
	ctx := context.Background()
	const numSessions = 5

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	// Create multiple sessions concurrently
	for i := 0; i < numSessions; i++ {
		wg.Add(1)
		go func(sessionID int) {
			defer wg.Done()

			// Create unique agent per session
			llm := &mockStreamingChatModel{
				responses:      []string{fmt.Sprintf("Response from agent %d", sessionID)},
				streamingDelay: 5 * time.Millisecond,
			}

			baseAgent, err := agents.NewBaseAgent(fmt.Sprintf("agent-%d", sessionID), llm, nil, agents.WithStreaming(true))
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}

			agent, ok := baseAgent.(iface.StreamingAgent)
			if !ok {
				mu.Lock()
				errors = append(errors, fmt.Errorf("agent %d does not implement StreamingAgent", sessionID))
				mu.Unlock()
				return
			}

			agentConfig := &schema.AgentConfig{
				Name:            fmt.Sprintf("agent-%d", sessionID),
				LLMProviderName: "mock",
			}

			sttProvider := &mockStreamingSTTProvider{
				transcript: fmt.Sprintf("Session %d transcript", sessionID),
			}
			ttsProvider := &mockStreamingTTSProvider{}

			voiceSession, err := session.NewVoiceSession(ctx,
				session.WithSTTProvider(sttProvider),
				session.WithTTSProvider(ttsProvider),
				session.WithAgentInstance(agent, agentConfig),
			)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}

			// Start session
			err = voiceSession.Start(ctx)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}

			// Verify session is listening
			assert.Equal(t, "listening", string(voiceSession.GetState()))

			// Stop session
			err = voiceSession.Stop(ctx)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}
		}(i)
	}

	// Wait for all sessions to complete
	wg.Wait()

	// Verify no errors occurred
	mu.Lock()
	defer mu.Unlock()
	assert.Empty(t, errors, "No errors should occur during concurrent session creation")
}

// TestAgentsVoice_Concurrent_StreamingExecution tests concurrent streaming execution.
func TestAgentsVoice_Concurrent_StreamingExecution(t *testing.T) {
	ctx := context.Background()
	const numStreams = 3

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	// Create a single agent
	llm := &mockStreamingChatModel{
		responses:      []string{"Response", "chunk", "one", "Response", "chunk", "two"},
		streamingDelay: 10 * time.Millisecond,
	}

	baseAgent, err := agents.NewBaseAgent("concurrent-agent", llm, nil, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok, "Agent should implement StreamingAgent")

	// Execute multiple streams concurrently
	for i := 0; i < numStreams; i++ {
		wg.Add(1)
		go func(streamID int) {
			defer wg.Done()

			inputs := map[string]any{
				"input": fmt.Sprintf("Request %d", streamID),
			}

			chunkChan, err := agent.StreamExecute(ctx, inputs)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}

			// Consume chunks
			chunkCount := 0
			for range chunkChan {
				chunkCount++
			}

			// Verify we received chunks
			if chunkCount == 0 {
				mu.Lock()
				errors = append(errors, fmt.Errorf("stream %d received no chunks", streamID))
				mu.Unlock()
			}
		}(i)
	}

	// Wait for all streams to complete
	wg.Wait()

	// Verify no errors occurred
	mu.Lock()
	defer mu.Unlock()
	assert.Empty(t, errors, "No errors should occur during concurrent streaming")
}
