package react

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// mockChatModel implements ChatModel interface for testing
type mockChatModel struct {
	response      string
	shouldError   bool
	generateCount int
}

func (m *mockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	if m.shouldError {
		return nil, errors.New("mock chat model error")
	}
	return schema.NewAIMessage(m.response), nil
}

func (m *mockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i := range inputs {
		result, err := m.Invoke(ctx, inputs[i], options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	ch <- schema.NewAIMessage(m.response)
	close(ch)
	return ch, nil
}

func (m *mockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	m.generateCount++
	if m.shouldError {
		return nil, errors.New("mock chat model generate error")
	}
	return schema.NewAIMessage(m.response), nil
}

func (m *mockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
	ch := make(chan llmsiface.AIMessageChunk, 1)
	ch <- llmsiface.AIMessageChunk{
		Content: m.response,
	}
	close(ch)
	return ch, nil
}

func (m *mockChatModel) BindTools(toolsToBind []iface.Tool) llmsiface.ChatModel {
	return m
}

func (m *mockChatModel) GetModelName() string {
	return "mock-chat-model"
}

func (m *mockChatModel) GetProviderName() string {
	return "mock-provider"
}

func (m *mockChatModel) CheckHealth() map[string]interface{} {
	return map[string]interface{}{
		"status": "healthy",
		"name":   "mock-chat-model",
	}
}

// mockTool implements Tool interface for testing
type mockTool struct {
	name        string
	description string
	result      any
	shouldError bool
	callCount   int
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Definition() iface.ToolDefinition {
	return tools.ToolDefinition{
		Name:        m.name,
		Description: m.description,
		InputSchema: map[string]interface{}{"type": "string"},
	}
}

func (m *mockTool) Execute(ctx context.Context, input any) (any, error) {
	m.callCount++
	if m.shouldError {
		return nil, errors.New("mock tool error")
	}
	return m.result, nil
}

func (m *mockTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
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

func TestNewReActAgent(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		llm       llmsiface.ChatModel
		tools     []iface.Tool
		prompt    interface{}
		opts      []iface.Option
		wantErr   bool
	}{
		{
			name:      "valid react agent creation",
			agentName: "test-react-agent",
			llm:       &mockChatModel{response: "test response"},
			tools:     []iface.Tool{&mockTool{name: "test-tool", description: "test tool"}},
			prompt:    "Test prompt template",
			opts:      []iface.Option{},
			wantErr:   false,
		},
		{
			name:      "nil chat model",
			agentName: "test-agent",
			llm:       nil,
			tools:     []iface.Tool{&mockTool{name: "test-tool", description: "test tool"}},
			prompt:    "Test prompt",
			wantErr:   true,
		},
		{
			name:      "nil tools",
			agentName: "test-agent",
			llm:       &mockChatModel{response: "test"},
			tools:     nil,
			prompt:    "Test prompt",
			wantErr:   false, // ReActAgent allows nil tools
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewReActAgent(tt.agentName, tt.llm, tt.tools, tt.prompt, tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if agent == nil {
				t.Error("Expected non-nil agent")
			}
		})
	}
}

func TestReActAgent_Plan(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		shouldError  bool
		expectFinish bool
		expectedTool string
	}{
		{
			name:         "final answer response",
			response:     "Final Answer: The answer is 42",
			shouldError:  false,
			expectFinish: true,
		},
		{
			name:         "action response",
			response:     "Action: calculator\nAction Input: {\"expression\": \"2 + 2\"}",
			shouldError:  false,
			expectFinish: false,
			expectedTool: "calculator",
		},
		{
			name:        "llm error",
			response:    "",
			shouldError: true,
		},
		{
			name:        "malformed response",
			response:    "This is not a valid ReAct response",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &mockChatModel{
				response:    tt.response,
				shouldError: tt.shouldError,
			}
			mockTool := &mockTool{name: "calculator", description: "Calculator tool", result: "4"}
			tools := []iface.Tool{mockTool}

			agent, err := NewReActAgent("test-agent", mockLLM, tools, "Test prompt")
			if err != nil {
				t.Fatalf("Failed to create agent: %v", err)
			}

			ctx := context.Background()
			inputs := map[string]any{"input": "test query"}
			steps := []iface.IntermediateStep{}

			action, finish, err := agent.Plan(ctx, steps, inputs)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectFinish {
				if finish.ReturnValues == nil {
					t.Error("Expected finish with return values")
				}
				if action.Tool != "" {
					t.Error("Expected no action when finish is returned")
				}
			} else {
				if finish.ReturnValues != nil {
					t.Error("Expected no finish when action is returned")
				}
				if action.Tool != tt.expectedTool {
					t.Errorf("Expected tool '%s', got '%s'", tt.expectedTool, action.Tool)
				}
			}
		})
	}
}

func TestReActAgent_ConstructScratchpad(t *testing.T) {
	mockLLM := &mockChatModel{response: "test"}
	tools := []iface.Tool{&mockTool{name: "test-tool", description: "test tool"}}

	agent, err := NewReActAgent("test-agent", mockLLM, tools, "Test prompt")
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	steps := []iface.IntermediateStep{
		{
			Action: iface.AgentAction{
				Tool:      "calculator",
				ToolInput: map[string]any{"expression": "2 + 2"},
				Log:       "Calculating sum",
			},
			Observation: "Result: 4",
		},
	}

	scratchpad := agent.constructScratchpad(steps)

	if scratchpad == "" {
		t.Error("Expected non-empty scratchpad")
	}

	if !strings.Contains(scratchpad, "Step 1:") {
		t.Error("Expected 'Step 1:' in scratchpad")
	}

	if !strings.Contains(scratchpad, "calculator") {
		t.Error("Expected 'calculator' in scratchpad")
	}

	if !strings.Contains(scratchpad, "Result: 4") {
		t.Error("Expected observation in scratchpad")
	}
}

func TestReActAgent_FormatPrompt(t *testing.T) {
	mockLLM := &mockChatModel{response: "test"}
	tools := []iface.Tool{&mockTool{name: "test-tool", description: "test tool"}}

	agent, err := NewReActAgent("test-agent", mockLLM, tools, "Input: {input}\nScratchpad: {agent_scratchpad}\nTools: {tools}")
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	inputs := map[string]any{
		"input":            "test query",
		"agent_scratchpad": "test scratchpad",
		"tools":            "test tools",
	}

	formatted := agent.formatPrompt(inputs)

	if !strings.Contains(formatted, "test query") {
		t.Error("Expected input to be formatted")
	}

	if !strings.Contains(formatted, "test scratchpad") {
		t.Error("Expected scratchpad to be formatted")
	}

	if !strings.Contains(formatted, "test tools") {
		t.Error("Expected tools to be formatted")
	}
}

func TestReActAgent_ParseResponse(t *testing.T) {
	mockLLM := &mockChatModel{response: "test"}
	tools := []iface.Tool{
		&mockTool{name: "calculator", description: "Calculator tool"},
	}

	agent, err := NewReActAgent("test-agent", mockLLM, tools, "Test prompt")
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	tests := []struct {
		name         string
		response     string
		expectFinish bool
		expectAction bool
		expectError  bool
	}{
		{
			name:         "final answer",
			response:     "Final Answer: The answer is 42",
			expectFinish: true,
			expectAction: false,
			expectError:  false,
		},
		{
			name:         "action with json input",
			response:     "Action: calculator\nAction Input: {\"expression\": \"2 + 2\"}",
			expectFinish: false,
			expectAction: true,
			expectError:  false,
		},
		{
			name:        "unknown tool",
			response:    "Action: unknown_tool\nAction Input: {}",
			expectError: true,
		},
		{
			name:        "malformed action",
			response:    "Action: calculator", // Missing Action Input
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, finish, err := agent.parseResponse(tt.response)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectFinish {
				if finish.ReturnValues == nil {
					t.Error("Expected finish with return values")
				}
			}

			if tt.expectAction {
				if action.Tool == "" {
					t.Error("Expected action with tool name")
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkNewReActAgent(b *testing.B) {
	mockLLM := &mockChatModel{response: "test"}
	tools := []iface.Tool{&mockTool{name: "test-tool", description: "test tool"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewReActAgent("bench-agent", mockLLM, tools, "Test prompt")
	}
}

func BenchmarkReActAgent_ConstructScratchpad(b *testing.B) {
	mockLLM := &mockChatModel{response: "test"}
	tools := []iface.Tool{&mockTool{name: "test-tool", description: "test tool"}}

	agent, _ := NewReActAgent("bench-agent", mockLLM, tools, "Test prompt")

	steps := []iface.IntermediateStep{
		{
			Action: iface.AgentAction{
				Tool:      "calculator",
				ToolInput: map[string]any{"expression": "2 + 2"},
				Log:       "Calculating sum",
			},
			Observation: "Result: 4",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agent.constructScratchpad(steps)
	}
}
