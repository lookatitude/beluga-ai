package mock

import (
	"context"
	"errors"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockLLM is a mock implementation of the llms.LLM interface for testing.
type MockLLM struct {
	Cfg       config.MockLLMConfig
	responses []string
	toolCalls []schema.ToolCall // Simplified for now, might need more complex structure
	err       error
	callCount int
}

// NewMockLLM creates a new MockLLM.
func NewMockLLM(cfg config.MockLLMConfig) (*MockLLM, error) {
	m := &MockLLM{
		Cfg:       cfg,
		responses: cfg.Responses,
	}
	if cfg.ExpectedError != "" {
		m.err = errors.New(cfg.ExpectedError)
	}
	// Convert map[string]interface{} to schema.ToolCall if provided in config
	for _, tcMap := range cfg.ToolCalls {
		// This is a simplified conversion. A more robust one would handle types and errors.
		toolCall := schema.ToolCall{}
		if id, ok := tcMap["id"].(string); ok {
			toolCall.ID = id
		}
		if typeStr, ok := tcMap["type"].(string); ok {
			toolCall.Type = typeStr
		}
		if funcMap, ok := tcMap["function"].(map[string]interface{}); ok {
			if name, ok := funcMap["name"].(string); ok {
				toolCall.Function.Name = name
			}
			if args, ok := funcMap["arguments"].(string); ok {
				toolCall.Function.Arguments = args
			}
		}
		m.toolCalls = append(m.toolCalls, toolCall)
	}

	return m, nil
}

// Invoke returns a predefined response or error.
func (m *MockLLM) Invoke(ctx context.Context, prompt string, callOptions ...schema.LLMOption) (string, error) {
	m.callCount++
	if m.err != nil {
		return "", m.err
	}
	if len(m.responses) > 0 {
		response := m.responses[0]
		if len(m.responses) > 1 {
			m.responses = m.responses[1:] // Consume the response
		} else if len(m.responses) == 1 {
			// Keep the last response for subsequent calls if only one is left
		}
		return response, nil
	}
	return fmt.Sprintf("Mock response to: %s", prompt), nil
}

// Chat returns a predefined ChatMessage response or error.
func (m *MockLLM) Chat(ctx context.Context, messages []schema.Message, callOptions ...schema.LLMOption) (schema.Message, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}

	var responseContent string
	if len(m.responses) > 0 {
		responseContent = m.responses[0]
		if len(m.responses) > 1 {
			m.responses = m.responses[1:] // Consume the response
		} else if len(m.responses) == 1 {
			// Keep the last response for subsequent calls if only one is left
		}
	} else {
		responseContent = "Mock chat response"
	}

	// Check if there are tool calls to return
	if len(m.toolCalls) > 0 {
		tc := m.toolCalls[0]
		if len(m.toolCalls) > 1 {
			m.toolCalls = m.toolCalls[1:] // Consume the tool call
		} else if len(m.toolCalls) == 1 {
			// Keep the last tool call for subsequent calls if only one is left
		}
		return &schema.ChatMessage{
			BaseMessage: schema.BaseMessage{Content: responseContent},
			Role:        schema.RoleAssistant,
			ToolCalls:   []schema.ToolCall{tc},
		}, nil
	}

	return schema.NewAIMessage(responseContent), nil
}

// GetModelName returns the configured model name.
func (m *MockLLM) GetModelName() string {
	return m.Cfg.ModelName
}

// GetProviderName returns the provider name for the mock LLM.
func (m *MockLLM) GetProviderName() string {
	return "mock"
}

// GetDefaultCallOptions returns default call options (currently empty for mock).
func (m *MockLLM) GetDefaultCallOptions() []schema.LLMOption {
	// For a mock, we might not have complex default options
	// or they could be configured via MockLLMConfig if needed.
	return []schema.LLMOption{}
}

// SetResponses allows setting responses for testing.
func (m *MockLLM) SetResponses(responses []string) {
	m.responses = responses
}

// SetError allows setting an error for testing.
func (m *MockLLM) SetError(err error) {
	m.err = err
}

// SetToolCalls allows setting tool calls for testing.
func (m *MockLLM) SetToolCalls(toolCalls []schema.ToolCall) {
	m.toolCalls = toolCalls
}

// CallCount returns how many times Invoke or Chat was called.
func (m *MockLLM) GetCallCount() int {
	return m.callCount
}

// Ensure MockLLM implements the LLM interface
var _ llms.LLM = (*MockLLM)(nil)

