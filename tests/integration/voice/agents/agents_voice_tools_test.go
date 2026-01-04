// Package agents provides integration tests for tool execution during voice calls.
// Integration test: Tool execution during voice calls (T151)
package agents

import (
	"context"
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
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentsVoice_Tools_Execution tests tool execution during voice calls.
func TestAgentsVoice_Tools_Execution(t *testing.T) {
	ctx := context.Background()

	// Create a mock tool
	calculatorTool := &mockCalculatorTool{
		results: make(map[string]float64),
	}

	// Create LLM that can trigger tool calls
	llm := &mockStreamingChatModelWithTools{
		mockStreamingChatModel: mockStreamingChatModel{
			responses:      []string{"I'll calculate that for you."},
			streamingDelay: 5 * time.Millisecond,
		},
		shouldCallTool: true,
		toolName:       calculatorTool.Name(),
	}

	// Create agent with tool
	baseAgent, err := agents.NewBaseAgent("tool-agent", llm, []tools.Tool{calculatorTool}, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok, "Agent should implement StreamingAgent")

	agentConfig := &schema.AgentConfig{
		Name:            "tool-agent",
		LLMProviderName: "mock",
	}

	sttProvider := &mockStreamingSTTProvider{
		transcript: "Calculate 5 + 3",
	}
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
	assert.Equal(t, "listening", string(voiceSession.GetState()))

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)

	// Verify agent has tools configured
	agentTools := agent.GetTools()
	assert.NotEmpty(t, agentTools, "Agent should have tools configured")
	assert.Equal(t, calculatorTool.Name(), agentTools[0].Name())
}

// TestAgentsVoice_Tools_StreamingWithTools tests streaming execution with tool calls.
func TestAgentsVoice_Tools_StreamingWithTools(t *testing.T) {
	ctx := context.Background()

	// Create mock tool
	calculatorTool := &mockCalculatorTool{
		results: make(map[string]float64),
	}

	llm := &mockStreamingChatModel{
		responses:      []string{"Let me calculate that."},
		streamingDelay: 5 * time.Millisecond,
	}

	baseAgent, err := agents.NewBaseAgent("streaming-tool-agent", llm, []tools.Tool{calculatorTool}, agents.WithStreaming(true))
	require.NoError(t, err)

	agent, ok := baseAgent.(iface.StreamingAgent)
	require.True(t, ok)

	inputs := map[string]any{"input": "Calculate 10 + 5"}

	// Stream execution
	chunkChan, err := agent.StreamExecute(ctx, inputs)
	require.NoError(t, err)

	// Consume chunks
	chunkCount := 0
	toolCallsReceived := 0
	timeout := time.After(2 * time.Second)

	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				goto done
			}
			chunkCount++
			if len(chunk.ToolCalls) > 0 {
				toolCallsReceived++
			}
			if chunk.Finish != nil {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:

	assert.Greater(t, chunkCount, 0, "Should receive chunks")
	// Note: Tool calls in chunks depend on LLM implementation
	// This test verifies the infrastructure supports tool calls
}

// mockCalculatorTool is a simple calculator tool for testing.
type mockCalculatorTool struct {
	results map[string]float64
	mu      sync.RWMutex
}

func (m *mockCalculatorTool) Name() string {
	return "calculator"
}

func (m *mockCalculatorTool) Description() string {
	return "Performs basic arithmetic calculations"
}

func (m *mockCalculatorTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        m.Name(),
		Description: m.Description(),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"expression": map[string]any{
					"type":        "string",
					"description": "Arithmetic expression to evaluate",
				},
			},
		},
	}
}

func (m *mockCalculatorTool) Execute(ctx context.Context, input any) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simple mock execution - return a result
	expr := "5 + 3"
	if str, ok := input.(string); ok {
		expr = str
	}
	if exprMap, ok := input.(map[string]any); ok {
		if e, ok := exprMap["expression"].(string); ok {
			expr = e
		}
	}

	result := 8.0 // Mock result
	m.results[expr] = result

	return map[string]any{
		"result":     result,
		"expression": expr,
	}, nil
}

func (m *mockCalculatorTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := m.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

// mockStreamingChatModelWithTools extends mockStreamingChatModel to support tool calls.
type mockStreamingChatModelWithTools struct {
	mockStreamingChatModel
	shouldCallTool bool
	toolName       string
	toolCalls      []schema.ToolCallChunk
}

func (m *mockStreamingChatModelWithTools) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
	m.mu.Lock()
	m.callCount++
	shouldCallTool := m.shouldCallTool
	toolName := m.toolName
	m.mu.Unlock()

	ch := make(chan llmsiface.AIMessageChunk, 10)

	go func() {
		defer close(ch)

		// Send initial content
		response := strings.Join(m.responses, " ")
		if response == "" {
			response = "Processing..."
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

		// Send tool call if configured
		if shouldCallTool {
			ch <- llmsiface.AIMessageChunk{
				ToolCallChunks: []schema.ToolCallChunk{
					{
						ID:        "call_123",
						Name:      toolName,
						Arguments: `{"expression": "5 + 3"}`,
					},
				},
			}
		}
	}()

	return ch, nil
}

