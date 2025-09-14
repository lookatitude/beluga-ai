package base

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockLLM is a test implementation of the LLM interface
type MockLLM struct {
	response string
}

func (m *MockLLM) Invoke(ctx context.Context, prompt string, callOptions ...schema.LLMOption) (string, error) {
	return m.response, nil
}

func (m *MockLLM) GetModelName() string {
	return "mock-model"
}

func (m *MockLLM) GetProviderName() string {
	return "mock-provider"
}

// MockMemory is a test implementation of the Memory interface
type MockMemory struct {
	data map[string]interface{}
}

func (m *MockMemory) LoadVariables(inputs map[string]interface{}) (map[string]interface{}, error) {
	return m.data, nil
}

func (m *MockMemory) SaveContext(inputs, outputs map[string]interface{}) error {
	return nil
}

func (m *MockMemory) Clear() error {
	m.data = make(map[string]interface{})
	return nil
}

func (m *MockMemory) MemoryVariables() []string {
	return []string{}
}

func TestNewBaseAgent(t *testing.T) {
	config := schema.AgentConfig{
		Name: "test-agent",
	}
	mockLLM := &MockLLM{response: "test response"}
	mockMemory := &MockMemory{data: make(map[string]interface{})}

	agent := NewBaseAgent(config, mockLLM, []tools.Tool{}, mockMemory)

	if agent.GetConfig().Name != "test-agent" {
		t.Errorf("Expected agent name to be 'test-agent', got %q", agent.GetConfig().Name)
	}

	if agent.GetLLM().GetModelName() != "mock-model" {
		t.Errorf("Expected LLM model name to be 'mock-model', got %q", agent.GetLLM().GetModelName())
	}

	if len(agent.GetTools()) != 0 {
		t.Errorf("Expected no tools, got %d", len(agent.GetTools()))
	}
}

func TestBaseAgent_GetConfig(t *testing.T) {
	config := schema.AgentConfig{Name: "test-agent"}
	agent := NewBaseAgent(config, &MockLLM{}, []tools.Tool{}, &MockMemory{})

	retrievedConfig := agent.GetConfig()
	if retrievedConfig.Name != "test-agent" {
		t.Errorf("Expected config name to be 'test-agent', got %q", retrievedConfig.Name)
	}
}

func TestBaseAgent_GetLLM(t *testing.T) {
	mockLLM := &MockLLM{response: "test"}
	agent := NewBaseAgent(schema.AgentConfig{}, mockLLM, []tools.Tool{}, &MockMemory{})

	llm := agent.GetLLM()
	if llm.GetModelName() != "mock-model" {
		t.Errorf("Expected LLM model name to be 'mock-model', got %q", llm.GetModelName())
	}
}

func TestBaseAgent_GetMemory(t *testing.T) {
	mockMemory := &MockMemory{data: make(map[string]interface{})}
	agent := NewBaseAgent(schema.AgentConfig{}, &MockLLM{}, []tools.Tool{}, mockMemory)

	memory := agent.GetMemory()
	if memory == nil {
		t.Error("Expected non-nil memory")
	}
}

func TestBaseAgent_InputKeys(t *testing.T) {
	agent := NewBaseAgent(schema.AgentConfig{}, &MockLLM{}, []tools.Tool{}, &MockMemory{})

	keys := agent.InputKeys()
	expected := []string{"input"}

	if len(keys) != len(expected) {
		t.Errorf("Expected %d input keys, got %d", len(expected), len(keys))
		return
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("Expected input key[%d] to be %q, got %q", i, expected[i], key)
		}
	}
}

func TestBaseAgent_OutputKeys(t *testing.T) {
	agent := NewBaseAgent(schema.AgentConfig{}, &MockLLM{}, []tools.Tool{}, &MockMemory{})

	keys := agent.OutputKeys()
	expected := []string{"output"}

	if len(keys) != len(expected) {
		t.Errorf("Expected %d output keys, got %d", len(expected), len(keys))
		return
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("Expected output key[%d] to be %q, got %q", i, expected[i], key)
		}
	}
}

func TestBaseAgent_Plan(t *testing.T) {
	agent := NewBaseAgent(schema.AgentConfig{}, &MockLLM{}, []tools.Tool{}, &MockMemory{})

	steps, err := agent.Plan(context.Background(), map[string]interface{}{}, []schema.Step{})
	if err == nil {
		t.Error("Expected Plan to return an error (not implemented)")
	}

	if steps != nil {
		t.Error("Expected Plan to return nil steps when not implemented")
	}
}

func TestBaseAgent_Execute(t *testing.T) {
	agent := NewBaseAgent(schema.AgentConfig{}, &MockLLM{}, []tools.Tool{}, &MockMemory{})

	result, err := agent.Execute(context.Background(), []schema.Step{})
	if err == nil {
		t.Error("Expected Execute to return an error (not implemented)")
	}

	if result != (schema.FinalAnswer{}) {
		t.Error("Expected Execute to return empty FinalAnswer when not implemented")
	}
}
