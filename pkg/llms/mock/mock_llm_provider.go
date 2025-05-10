package mock

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockLLM is a mock implementation of the llms.LLM interface for testing purposes.
type MockLLM struct {
	Responses      map[string]string // A map of prompt prefixes to canned responses
	InvokeCount    int
	ExpectedError  error // If set, Invoke will return this error
	ModelName      string
	ProviderName   string
	LastPrompt     string
	LastCallOptions []schema.LLMOption
}

// NewMockLLM creates a new MockLLM instance.
func NewMockLLM(responses map[string]string, expectedError error) *MockLLM {
	return &MockLLM{
		Responses:     responses,
		ExpectedError: expectedError,
		ModelName:     "mock-model",
		ProviderName:  "mock",
	}
}

// Invoke returns a canned response based on the prompt or an error if configured.
func (m *MockLLM) Invoke(ctx context.Context, prompt string, callOptions ...schema.LLMOption) (string, error) {
	m.InvokeCount++
	m.LastPrompt = prompt
	m.LastCallOptions = callOptions

	if m.ExpectedError != nil {
		return "", m.ExpectedError
	}

	// Simple prefix matching for responses
	for p, resp := range m.Responses {
		if len(prompt) >= len(p) && prompt[:len(p)] == p {
			return resp, nil
		}
	}
	// Default response if no match found
	if defaultResp, ok := m.Responses["DEFAULT"]; ok {
		return defaultResp, nil
	}

	return fmt.Sprintf("Mock response for prompt: %s", prompt), nil
}

// GetModelName returns the model name.
func (m *MockLLM) GetModelName() string {
	return m.ModelName
}

// GetProviderName returns the provider name.
func (m *MockLLM) GetProviderName() string {
	return m.ProviderName
}

// Ensure MockLLM implements the llms.LLM interface.
var _ llms.LLM = (*MockLLM)(nil)

