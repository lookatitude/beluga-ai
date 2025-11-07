package base

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// MockLLM is a test implementation of the LLM interface
type MockLLM struct {
	response any
	err      error
}

func (m *MockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *MockLLM) GetModelName() string {
	return "mock-model"
}

func (m *MockLLM) GetProviderName() string {
	return "mock-provider"
}

// MockTool is a test implementation of the Tool interface
type MockTool struct {
	name        string
	description string
	result      interface{}
	err         error
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        m.name,
		Description: m.description,
	}
}

func (m *MockTool) Execute(ctx context.Context, input any) (any, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func (m *MockTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewBaseAgent(t *testing.T) {
	tests := []struct {
		name        string
		agentName   string
		llm         llmsiface.LLM
		tools       []tools.Tool
		opts        []iface.Option
		expectError bool
	}{
		{
			name:        "valid agent creation",
			agentName:   "test-agent",
			llm:         &MockLLM{response: "test response"},
			tools:       []tools.Tool{&MockTool{name: "test-tool", description: "test tool"}},
			opts:        []iface.Option{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewBaseAgent(tt.agentName, tt.llm, tt.tools, tt.opts...)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if agent == nil {
				t.Error("Expected non-nil agent")
				return
			}

			// Verify agent properties
			if agent.GetLLM() != tt.llm {
				t.Error("LLM not set correctly")
			}

			if len(agent.GetTools()) != len(tt.tools) {
				t.Errorf("Expected %d tools, got %d", len(tt.tools), len(agent.GetTools()))
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestBaseAgent_Initialize(t *testing.T) {
	llm := &MockLLM{response: "test"}
	tools := []tools.Tool{&MockTool{name: "test-tool", description: "test tool"}}
	agent, err := NewBaseAgent("test-agent", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	config := map[string]interface{}{
		"max_retries": 5,
		"timeout":     "30s",
	}

	err = agent.Initialize(config)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	// Verify state changed to ready
	if agent.GetState() != iface.StateReady {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("Expected state Ready, got %v", agent.GetState())
	}
}

func TestBaseAgent_GetConfig(t *testing.T) {
	llm := &MockLLM{response: "test"}
	tools := []tools.Tool{}
	agent, err := NewBaseAgent("test-agent", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	config := agent.GetConfig()
	if config.Name != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", config.Name)
	}
}

func TestBaseAgent_GetTools(t *testing.T) {
	llm := &MockLLM{response: "test"}
	tool1 := &MockTool{name: "tool1", description: "first tool"}
	tool2 := &MockTool{name: "tool2", description: "second tool"}
	tools := []tools.Tool{tool1, tool2}

	agent, err := NewBaseAgent("test-agent", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	retrievedTools := agent.GetTools()
	if len(retrievedTools) != 2 {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("Expected 2 tools, got %d", len(retrievedTools))
	}

	if retrievedTools[0].Name() != "tool1" {
		t.Errorf("Expected first tool name 'tool1', got '%s'", retrievedTools[0].Name())
	}
}

func TestBaseAgent_CheckHealth(t *testing.T) {
	llm := &MockLLM{response: "test"}
	tools := []tools.Tool{&MockTool{name: "test-tool", description: "test tool"}}
	agent, err := NewBaseAgent("test-agent", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	health := agent.CheckHealth()

	if health["name"] != "test-agent" {
		t.Errorf("Expected name 'test-agent', got '%v'", health["name"])
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	if health["state"] != iface.StateInitializing {
		t.Errorf("Expected initial state 'initializing', got '%v'", health["state"])
	}

	if health["tools_count"] != 1 {
		t.Errorf("Expected 1 tool, got '%v'", health["tools_count"])
	}
}

func TestBaseAgent_EventHandling(t *testing.T) {
	llm := &MockLLM{response: "test"}
	tools := []tools.Tool{}
	agent, err := NewBaseAgent("test-agent", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	eventReceived := false
	var receivedPayload interface{}

	handler := func(payload interface{}) error {
		eventReceived = true
		receivedPayload = payload
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	agent.RegisterEventHandler("test_event", handler)
	agent.EmitEvent("test_event", "test_payload")

	if !eventReceived {
		t.Error("Event handler was not called")
	}

	if receivedPayload != "test_payload" {
		t.Errorf("Expected payload 'test_payload', got '%v'", receivedPayload)
	}
}

func TestBaseAgent_Plan(t *testing.T) {
	llm := &MockLLM{response: "test"}
	tools := []tools.Tool{}
	agent, err := NewBaseAgent("test-agent", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}

	ctx := context.Background()
	inputs := map[string]interface{}{"input": "test input"}
	steps := []iface.IntermediateStep{}

	_, _, err = agent.Plan(ctx, steps, inputs)

	// Base agent should return an error since Plan is not implemented
	if err == nil {
		t.Error("Expected Plan to return an error (not implemented)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func BenchmarkBaseAgent_CheckHealth(b *testing.B) {
	llm := &MockLLM{response: "test"}
	tools := []tools.Tool{&MockTool{name: "test-tool", description: "test tool"}}
	agent, err := NewBaseAgent("test-agent", llm, tools)
	if err != nil {
		b.Fatalf("Failed to create agent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agent.CheckHealth()
	}
}

func BenchmarkBaseAgent_EventEmission(b *testing.B) {
	llm := &MockLLM{response: "test"}
	tools := []tools.Tool{}
	agent, err := NewBaseAgent("test-agent", llm, tools)
	if err != nil {
		b.Fatalf("Failed to create agent: %v", err)
	}

	handler := func(payload interface{}) error {
		return nil
	}

	agent.RegisterEventHandler("benchmark_event", handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent.EmitEvent("benchmark_event", i)
	}
}
