// Package internal provides tests for agent integration in session lifecycle.
// T144-T147: Unit tests for agent integration in session lifecycle
package internal

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T144: Add unit tests for agent integration in session lifecycle.
func TestSessionImpl_AgentIntegration_Lifecycle(t *testing.T) {
	ctx := context.Background()

	// Create mock streaming agent
	mockAgent := &mockStreamingAgentForSession{}
	agentConfig := &schema.AgentConfig{
		Name:            "test-agent",
		LLMProviderName: "mock",
	}

	// Create session with agent instance
	opts := &VoiceOptions{
		STTProvider:   &mockSTTProviderForAgentTest{},
		TTSProvider:   &mockTTSProviderForAgentTest{},
		AgentInstance: mockAgent,
		AgentConfig:   agentConfig,
	}
	config := &Config{
		Timeout: 30 * time.Minute,
	}

	session, err := NewVoiceSessionImpl(config, opts)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.NotNil(t, session.agentIntegration)
	assert.NotNil(t, session.streamingAgent)

	// Test Start - agent should be initialized
	err = session.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, sessioniface.SessionState("listening"), session.GetState())

	// Test Stop - agent should be cleaned up
	err = session.Stop(ctx)
	require.NoError(t, err)
	assert.Equal(t, sessioniface.SessionState("ended"), session.GetState())
}

// T145: Add unit tests for interruption handling integration.
func TestSessionImpl_AgentIntegration_Interruption(t *testing.T) {
	ctx := context.Background()

	mockAgent := &mockStreamingAgentForSession{}
	opts := &VoiceOptions{
		STTProvider:   &mockSTTProviderForAgentTest{},
		TTSProvider:   &mockTTSProviderForAgentTest{},
		AgentInstance: mockAgent,
		AgentConfig:   &schema.AgentConfig{Name: "test-agent"},
	}

	session, err := NewVoiceSessionImpl(&Config{Timeout: 30 * time.Minute}, opts)
	require.NoError(t, err)

	err = session.Start(ctx)
	require.NoError(t, err)

	// Simulate interruption by processing new audio while agent is streaming
	audio1 := []byte{1, 2, 3}
	err = session.ProcessAudio(ctx, audio1)
	require.NoError(t, err)

	// Process another audio immediately (should interrupt previous)
	audio2 := []byte{4, 5, 6}
	err = session.ProcessAudio(ctx, audio2)
	require.NoError(t, err)

	// Agent should handle interruption gracefully
	assert.NotEqual(t, sessioniface.SessionState("ended"), session.GetState())
}

// T146: Add unit tests for context preservation.
func TestSessionImpl_AgentIntegration_ContextPreservation(t *testing.T) {
	ctx := context.Background()

	mockAgent := &mockStreamingAgentForSession{}
	opts := &VoiceOptions{
		STTProvider:   &mockSTTProviderForAgentTest{},
		TTSProvider:   &mockTTSProviderForAgentTest{},
		AgentInstance: mockAgent,
		AgentConfig:   &schema.AgentConfig{Name: "test-agent"},
	}

	session, err := NewVoiceSessionImpl(&Config{Timeout: 30 * time.Minute}, opts)
	require.NoError(t, err)

	err = session.Start(ctx)
	require.NoError(t, err)

	// Process multiple audio inputs - context should be preserved
	audio1 := []byte{1, 2, 3}
	err = session.ProcessAudio(ctx, audio1)
	require.NoError(t, err)

	audio2 := []byte{4, 5, 6}
	err = session.ProcessAudio(ctx, audio2)
	require.NoError(t, err)

	// Agent instance should maintain context across interactions
	agentInstance := session.agentIntegration.GetAgentInstance()
	require.NotNil(t, agentInstance)
	assert.NotNil(t, agentInstance.GetContext())
}

// T147: Add unit tests for backward compatibility.
func TestSessionImpl_AgentIntegration_BackwardCompatibility(t *testing.T) {
	ctx := context.Background()

	// Test callback-based mode (backward compatibility)
	callback := func(ctx context.Context, transcript string) (string, error) {
		return "callback response", nil
	}

	opts := &VoiceOptions{
		STTProvider:   &mockSTTProviderForAgentTest{},
		TTSProvider:   &mockTTSProviderForAgentTest{},
		AgentCallback: callback,
	}

	session, err := NewVoiceSessionImpl(&Config{Timeout: 30 * time.Minute}, opts)
	require.NoError(t, err)
	assert.NotNil(t, session.agentIntegration)
	assert.Nil(t, session.streamingAgent) // No streaming agent in callback mode

	err = session.Start(ctx)
	require.NoError(t, err)

	// Process audio - should use callback
	// Note: ProcessAudio may be asynchronous, so we just verify the session was created correctly
	audio := []byte{1, 2, 3}
	err = session.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Verify session was created with callback mode (no streaming agent)
	assert.NotNil(t, session.agentIntegration)
	assert.Nil(t, session.streamingAgent) // No streaming agent in callback mode

	// The callback will be called asynchronously when transcript is processed
	// For this test, we just verify the setup is correct
	time.Sleep(100 * time.Millisecond) // Give time for async processing
	// Note: In a real scenario, the callback would be called when STT produces a transcript
}

// Mock implementations.
type mockStreamingAgentForSession struct{}

func (m *mockStreamingAgentForSession) StreamExecute(ctx context.Context, inputs map[string]any) (<-chan agentsiface.AgentStreamChunk, error) {
	ch := make(chan agentsiface.AgentStreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- agentsiface.AgentStreamChunk{Content: "response"}
	}()
	return ch, nil
}

func (m *mockStreamingAgentForSession) StreamPlan(ctx context.Context, intermediateSteps []agentsiface.IntermediateStep, inputs map[string]any) (<-chan agentsiface.AgentStreamChunk, error) {
	return m.StreamExecute(ctx, inputs)
}

func (m *mockStreamingAgentForSession) Plan(ctx context.Context, intermediateSteps []agentsiface.IntermediateStep, inputs map[string]any) (agentsiface.AgentAction, agentsiface.AgentFinish, error) {
	return agentsiface.AgentAction{}, agentsiface.AgentFinish{}, nil
}

func (m *mockStreamingAgentForSession) InputVariables() []string {
	return []string{"input"}
}

func (m *mockStreamingAgentForSession) OutputVariables() []string {
	return []string{"output"}
}

func (m *mockStreamingAgentForSession) GetTools() []tools.Tool {
	return nil
}

func (m *mockStreamingAgentForSession) GetConfig() schema.AgentConfig {
	return schema.AgentConfig{Name: "mock-agent"}
}

func (m *mockStreamingAgentForSession) GetLLM() llmsiface.LLM {
	return nil
}

func (m *mockStreamingAgentForSession) GetMetrics() agentsiface.MetricsRecorder {
	return nil
}

func (m *mockStreamingAgentForSession) Execute(ctx context.Context, inputs map[string]any, options ...agentsiface.Option) (any, error) {
	return "result", nil
}

type mockSTTProviderForAgentTest struct{}

func (m *mockSTTProviderForAgentTest) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "transcript", nil
}

func (m *mockSTTProviderForAgentTest) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return &mockStreamingSessionForAgentTest{}, nil
}

type mockStreamingSessionForAgentTest struct{}

func (m *mockStreamingSessionForAgentTest) SendAudio(ctx context.Context, audio []byte) error {
	return nil
}

func (m *mockStreamingSessionForAgentTest) ReceiveTranscript() <-chan voiceiface.TranscriptResult {
	ch := make(chan voiceiface.TranscriptResult, 1)
	ch <- voiceiface.TranscriptResult{Text: "transcript", IsFinal: true}
	close(ch)
	return ch
}

func (m *mockStreamingSessionForAgentTest) Close() error {
	return nil
}

type mockTTSProviderForAgentTest struct{}

func (m *mockTTSProviderForAgentTest) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	return []byte{1, 2, 3}, nil
}

func (m *mockTTSProviderForAgentTest) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return strings.NewReader(text), nil
}
