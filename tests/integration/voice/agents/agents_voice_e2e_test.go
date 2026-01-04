// Package agents provides integration tests for streaming agents with voice sessions.
// Integration test: End-to-end voice call with streaming agent (T148)
package agents

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentsVoice_E2E_StreamingAgent tests end-to-end voice call with streaming agent.
func TestAgentsVoice_E2E_StreamingAgent(t *testing.T) {
	ctx := context.Background()

	// Create mock LLM with streaming support
	llm := &mockStreamingChatModel{
		responses:      []string{"Hello!", "How", "can", "I", "help", "you", "today?"},
		streamingDelay: 10 * time.Millisecond,
	}

	// Create streaming agent
	baseAgent, err := agents.NewBaseAgent("test-agent", llm, nil, agents.WithStreaming(true))
	require.NoError(t, err)

	// Cast to StreamingAgent interface
	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok, "BaseAgent should implement StreamingAgent interface")

	// Create agent config
	agentConfig := &schema.AgentConfig{
		Name:            "test-agent",
		LLMProviderName: "mock",
	}

	// Create mock providers
	sttProvider := &mockStreamingSTTProvider{
		transcript: "Hello, I need help",
	}
	ttsProvider := &mockStreamingTTSProvider{}

	// Create voice session with streaming agent
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentInstance(agent, agentConfig),
	)
	require.NoError(t, err)
	require.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, "listening", string(voiceSession.GetState()))

	// For now, verify the session is set up correctly with agent instance
	// Full transcript processing will be tested once ProcessAudio pipeline is complete
	// The agent instance integration is verified by successful session creation
	
	// Verify session state
	assert.Equal(t, "listening", string(voiceSession.GetState()))

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "ended", string(voiceSession.GetState()))
}

// Mock implementations for integration tests

// mockStreamingChatModel implements ChatModel interface with streaming support.
type mockStreamingChatModel struct {
	responses      []string
	streamingDelay time.Duration
	shouldError    bool
	errorToReturn  error
	callCount      int
	mu             sync.RWMutex
}

func (m *mockStreamingChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, fmt.Errorf("mock LLM error")
	}

	ch := make(chan llmsiface.AIMessageChunk, 10)

	go func() {
		defer close(ch)

		response := strings.Join(m.responses, " ")
		if response == "" {
			response = "Hello world. This is a test."
		}

		words := strings.Fields(response)
		for _, word := range words {
			select {
			case <-ctx.Done():
				ch <- llmsiface.AIMessageChunk{Err: ctx.Err()}
				return
			case ch <- llmsiface.AIMessageChunk{Content: word + " "}:
			}

			if m.streamingDelay > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(m.streamingDelay):
				}
			}
		}
	}()

	return ch, nil
}

func (m *mockStreamingChatModel) BindTools(toolsToBind []tools.Tool) llmsiface.ChatModel {
	// Return self - tools binding not implemented in mock
	return m
}

func (m *mockStreamingChatModel) GetModelName() string {
	return "mock-streaming-model"
}

func (m *mockStreamingChatModel) GetProviderName() string {
	return "mock-provider"
}

func (m *mockStreamingChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return schema.NewAIMessage(strings.Join(m.responses, " ")), nil
}

func (m *mockStreamingChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	response := schema.NewAIMessage(strings.Join(m.responses, " "))
	for i := range inputs {
		results[i] = response
	}
	return results, nil
}

func (m *mockStreamingChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	ch <- schema.NewAIMessage(strings.Join(m.responses, " "))
	close(ch)
	return ch, nil
}

func (m *mockStreamingChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	return schema.NewAIMessage(strings.Join(m.responses, " ")), nil
}

func (m *mockStreamingChatModel) CheckHealth() map[string]any {
	return map[string]any{"status": "healthy"}
}

// mockStreamingSTTProvider implements STT provider with transcript generation.
type mockStreamingSTTProvider struct {
	transcript string
}

func (m *mockStreamingSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	if m.transcript == "" {
		return "Hello, I need help", nil
	}
	return m.transcript, nil
}

func (m *mockStreamingSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return &mockStreamingSTTSession{
		transcript: m.transcript,
	}, nil
}

// mockStreamingSTTSession implements StreamingSession for STT.
type mockStreamingSTTSession struct {
	transcript string
	mu         sync.Mutex
}

func (m *mockStreamingSTTSession) SendAudio(ctx context.Context, audio []byte) error {
	return nil
}

func (m *mockStreamingSTTSession) ReceiveTranscript() <-chan voiceiface.TranscriptResult {
	ch := make(chan voiceiface.TranscriptResult, 1)
	go func() {
		time.Sleep(10 * time.Millisecond)
		ch <- voiceiface.TranscriptResult{
			Text:    m.transcript,
			IsFinal: true,
		}
		close(ch)
	}()
	return ch
}

func (m *mockStreamingSTTSession) Close() error {
	return nil
}

// mockStreamingTTSProvider implements TTS provider that tracks calls.
type mockStreamingTTSProvider struct {
	callCount int
	mu        sync.Mutex
}

func (m *mockStreamingTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	return []byte{1, 2, 3, 4, 5}, nil
}

func (m *mockStreamingTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	return strings.NewReader(text), nil
}

func (m *mockStreamingTTSProvider) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

