package agents_test

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// mockLLM is a simple mock implementation of the LLM interface for testing
type mockLLM struct{}

func (m *mockLLM) Invoke(ctx context.Context, prompt string, options ...core.Option) (string, error) {
	return "Mock LLM response to: " + prompt, nil
}

func (m *mockLLM) GetModelName() string {
	return "mock-llm"
}

func (m *mockLLM) GetProviderName() string {
	return "mock-provider"
}

// mockChatModel is a simple mock implementation of the ChatModel interface for testing
type mockChatModel struct{}

func (m *mockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return schema.NewAIMessage("Mock chat model response"), nil
}

func (m *mockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = schema.NewAIMessage("Mock batch chat response")
	}
	return results, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	ch <- schema.NewAIMessage("Mock streaming chat response")
	close(ch)
	return ch, nil
}

func (m *mockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	return schema.NewAIMessage("Mock chat model response"), nil
}

func (m *mockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
	ch := make(chan llmsiface.AIMessageChunk, 1)
	ch <- llmsiface.AIMessageChunk{
		Content: "Mock streaming chat response",
	}
	close(ch)
	return ch, nil
}

func (m *mockChatModel) BindTools(toolsToBind []tools.Tool) llmsiface.ChatModel {
	return m
}

func (m *mockChatModel) GetModelName() string {
	return "mock-chat-model"
}

func (m *mockChatModel) CheckHealth() map[string]interface{} {
	return map[string]interface{}{
		"status": "healthy",
		"name":   "mock-chat-model",
	}
}

// mockTool is a simple mock implementation of the Tool interface for testing
type mockTool struct {
	name        string
	description string
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        m.name,
		Description: m.description,
		InputSchema: map[string]interface{}{"type": "string"},
	}
}

func (m *mockTool) Execute(ctx context.Context, input any) (any, error) {
	return "Mock tool result for input: " + input.(string), nil
}

func (m *mockTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		results[i] = "Mock batch tool result for input: " + input.(string)
	}
	return results, nil
}

func createMockTools() []tools.Tool {
	return []tools.Tool{
		&mockTool{name: "calculator", description: "A calculator tool"},
		&mockTool{name: "webSearch", description: "A web search tool"},
	}
}

func TestNewBaseAgent(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		llm       llmsiface.LLM
		tools     []tools.Tool
		opts      []iface.Option
		wantErr   bool
	}{
		{
			name:      "valid base agent creation",
			agentName: "test-agent",
			llm:       &mockLLM{},
			tools:     createMockTools(),
			opts:      []iface.Option{},
			wantErr:   false,
		},
		{
			name:      "agent with options",
			agentName: "test-agent-with-opts",
			llm:       &mockLLM{},
			tools:     createMockTools(),
			opts: []iface.Option{
				agents.WithMaxRetries(5),
				agents.WithTimeout(10 * time.Second),
			},
			wantErr: false,
		},
		{
			name:      "empty agent name",
			agentName: "",
			llm:       &mockLLM{},
			tools:     createMockTools(),
			opts:      []iface.Option{},
			wantErr:   false, // Base agent allows empty names
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := agents.NewBaseAgent(tt.agentName, tt.llm, tt.tools, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBaseAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && agent == nil {
				t.Error("NewBaseAgent() returned nil agent without error")
			}
		})
	}
}

func TestNewReActAgent(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		llm       llmsiface.ChatModel
		tools     []tools.Tool
		prompt    interface{}
		opts      []iface.Option
		wantErr   bool
	}{
		{
			name:      "valid react agent creation",
			agentName: "react-agent",
			llm:       &mockChatModel{},
			tools:     createMockTools(),
			prompt:    "Test prompt",
			opts:      []iface.Option{},
			wantErr:   false,
		},
		{
			name:      "react agent with options",
			agentName: "react-agent-with-opts",
			llm:       &mockChatModel{},
			tools:     createMockTools(),
			prompt:    "Test prompt with options",
			opts: []iface.Option{
				agents.WithMaxRetries(3),
				agents.WithTimeout(5 * time.Second),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := agents.NewReActAgent(tt.agentName, tt.llm, tt.tools, tt.prompt, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReActAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && agent == nil {
				t.Error("NewReActAgent() returned nil agent without error")
			}
		})
	}
}

func TestAgentFactory(t *testing.T) {
	config := agents.DefaultConfig()
	factory := agents.NewAgentFactory(config)

	ctx := context.Background()
	llm := &mockLLM{}
	tools := createMockTools()

	t.Run("CreateBaseAgent", func(t *testing.T) {
		agent, err := factory.CreateBaseAgent(ctx, "factory-base-agent", llm, tools)
		if err != nil {
			t.Errorf("AgentFactory.CreateBaseAgent() error = %v", err)
			return
		}
		if agent == nil {
			t.Error("AgentFactory.CreateBaseAgent() returned nil agent")
		}
	})

	t.Run("CreateReActAgent", func(t *testing.T) {
		chatLLM := &mockChatModel{}
		agent, err := factory.CreateReActAgent(ctx, "factory-react-agent", chatLLM, tools, "Test prompt")
		if err != nil {
			t.Errorf("AgentFactory.CreateReActAgent() error = %v", err)
			return
		}
		if agent == nil {
			t.Error("AgentFactory.CreateReActAgent() returned nil agent")
		}
	})
}

func TestNewAgentExecutor(t *testing.T) {
	executor := agents.NewAgentExecutor()
	if executor == nil {
		t.Error("NewAgentExecutor() returned nil")
	}
}

func TestNewAgentExecutorWithOptions(t *testing.T) {
	executor := agents.NewExecutorWithMaxIterations(10)
	if executor == nil {
		t.Error("NewExecutorWithMaxIterations() returned nil")
	}

	executor2 := agents.NewExecutorWithReturnIntermediateSteps(true)
	if executor2 == nil {
		t.Error("NewExecutorWithReturnIntermediateSteps() returned nil")
	}

	executor3 := agents.NewExecutorWithHandleParsingErrors(false)
	if executor3 == nil {
		t.Error("NewExecutorWithHandleParsingErrors() returned nil")
	}
}

func TestNewToolRegistry(t *testing.T) {
	registry := agents.NewToolRegistry()
	if registry == nil {
		t.Error("NewToolRegistry() returned nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := agents.DefaultConfig()
	if config == nil {
		t.Error("DefaultConfig() returned nil")
	}

	// Test default values
	if config.DefaultMaxRetries != 3 {
		t.Errorf("DefaultConfig().DefaultMaxRetries = %v, want %v", config.DefaultMaxRetries, 3)
	}
	if config.DefaultTimeout != 30*time.Second {
		t.Errorf("DefaultConfig().DefaultTimeout = %v, want %v", config.DefaultTimeout, 30*time.Second)
	}
	if !config.EnableMetrics {
		t.Error("DefaultConfig().EnableMetrics should be true by default")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *agents.Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  agents.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "negative max retries",
			config: &agents.Config{
				DefaultMaxRetries:    -1,
				DefaultRetryDelay:    2 * time.Second,
				DefaultTimeout:       30 * time.Second,
				DefaultMaxIterations: 15,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: &agents.Config{
				DefaultMaxRetries:    3,
				DefaultRetryDelay:    2 * time.Second,
				DefaultTimeout:       0,
				DefaultMaxIterations: 15,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := agents.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHealthCheck(t *testing.T) {
	llm := &mockLLM{}
	tools := createMockTools()
	agent, err := agents.NewBaseAgent("test-agent", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create test agent: %v", err)
	}

	// Initialize the agent to change its state to ready
	err = agent.Initialize(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to initialize test agent: %v", err)
	}

	status := agents.HealthCheck(agent)
	if status == nil {
		t.Error("HealthCheck() returned nil")
	}

	// Check that status contains expected fields
	if name, ok := status["name"].(string); !ok || name != "test-agent" {
		t.Errorf("HealthCheck() name = %v, want 'test-agent'", name)
	}

	if state, ok := status["state"].(iface.AgentState); !ok {
		t.Error("HealthCheck() missing or invalid state field")
	} else if state != iface.StateReady {
		t.Errorf("HealthCheck() state = %v, want %v", state, iface.StateReady)
	}
}

func TestListAgentStates(t *testing.T) {
	states := agents.ListAgentStates()
	expectedStates := []iface.AgentState{
		iface.StateInitializing,
		iface.StateReady,
		iface.StateRunning,
		iface.StatePaused,
		iface.StateError,
		iface.StateShutdown,
	}

	if len(states) != len(expectedStates) {
		t.Errorf("ListAgentStates() returned %d states, want %d", len(states), len(expectedStates))
	}

	for i, expected := range expectedStates {
		if i >= len(states) || states[i] != expected {
			t.Errorf("ListAgentStates()[%d] = %v, want %v", i, states[i], expected)
		}
	}
}

func TestGetAgentStateString(t *testing.T) {
	tests := []struct {
		state iface.AgentState
		want  string
	}{
		{iface.StateInitializing, "Initializing"},
		{iface.StateReady, "Ready"},
		{iface.StateRunning, "Running"},
		{iface.StatePaused, "Paused"},
		{iface.StateError, "Error"},
		{iface.StateShutdown, "Shutdown"},
		{iface.AgentState("unknown"), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := agents.GetAgentStateString(tt.state)
			if got != tt.want {
				t.Errorf("GetAgentStateString(%v) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

// Test agent as Runnable interface
func TestAgentAsRunnable(t *testing.T) {
	llm := &mockLLM{}
	tools := createMockTools()
	agent, err := agents.NewBaseAgent("runnable-agent", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create test agent: %v", err)
	}

	ctx := context.Background()
	input := map[string]any{"input": "test query"}

	// Test Invoke - expect error since BaseAgent doesn't implement executeWithInput
	_, err = agent.Invoke(ctx, input)
	if err == nil {
		t.Error("Agent.Invoke() expected error for unimplemented executeWithInput")
	}

	// Test Batch - expect error since BaseAgent doesn't implement executeWithInput
	inputs := []any{input, input}
	_, err = agent.Batch(ctx, inputs)
	if err == nil {
		t.Error("Agent.Batch() expected error for unimplemented executeWithInput")
	}

	// Test Stream - expect error since BaseAgent doesn't implement executeWithInput
	stream, err := agent.Stream(ctx, input)
	if err != nil {
		t.Errorf("Agent.Stream() unexpected error: %v", err)
	} else {
		select {
		case result := <-stream:
			if result == nil {
				t.Error("Agent.Stream() should return an error result")
			}
			// Should be an error
			if _, ok := result.(error); !ok {
				t.Error("Agent.Stream() expected error result")
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Agent.Stream() timed out waiting for result")
		}
	}
}
