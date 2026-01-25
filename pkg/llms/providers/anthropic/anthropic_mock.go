package anthropic

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// AdvancedMockAnthropicProvider provides a comprehensive mock implementation for testing Anthropic provider.
type AdvancedMockAnthropicProvider struct {
	errorToReturn     error
	mu                *sync.RWMutex
	modelName         string
	responses         []string
	boundTools        []tools.Tool
	callCount         int
	responseIndex     int
	simulateDelay     time.Duration
	rateLimitCount    int
	shouldError       bool
	simulateRateLimit bool
}

// NewAdvancedMockAnthropicProvider creates a new advanced mock with configurable behavior.
func NewAdvancedMockAnthropicProvider(modelName string) *AdvancedMockAnthropicProvider {
	mock := &AdvancedMockAnthropicProvider{
		mu:        &sync.RWMutex{},
		modelName: modelName,
		responses: []string{
			"This is a mock response from Anthropic.",
			"I understand your request and I'm here to help.",
			"Thank you for your question. Here's my response.",
		},
	}
	return mock
}

// Generate implements the ChatModel interface.
func (m *AdvancedMockAnthropicProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	simulateRateLimit := m.simulateRateLimit
	rateLimitCount := m.rateLimitCount
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	m.mu.Lock()
	if simulateRateLimit && rateLimitCount > 5 {
		m.mu.Unlock()
		return nil, llms.NewLLMError(ProviderName, llms.ErrCodeRateLimit, errors.New("rate limit exceeded"))
	}
	m.rateLimitCount++
	m.mu.Unlock()

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, llms.NewLLMError(ProviderName, llms.ErrCodeInvalidRequest, errors.New("mock error"))
	}

	m.mu.Lock()
	response := m.responses[m.responseIndex%len(m.responses)]
	m.responseIndex++
	m.mu.Unlock()

	return schema.NewAIMessage(response), nil
}

// StreamChat implements the ChatModel interface.
func (m *AdvancedMockAnthropicProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, llms.NewLLMError(ProviderName, llms.ErrCodeInvalidRequest, errors.New("mock error"))
	}

	streamChan := make(chan iface.AIMessageChunk, 10)

	go func() {
		defer close(streamChan)

		m.mu.Lock()
		response := m.responses[m.responseIndex%len(m.responses)]
		m.responseIndex++
		m.mu.Unlock()

		words := strings.Fields(response)
		for _, word := range words {
			if m.simulateDelay > 0 {
				time.Sleep(m.simulateDelay)
			}

			chunk := iface.AIMessageChunk{
				Content: word + " ",
			}

			select {
			case streamChan <- chunk:
			case <-ctx.Done():
				streamChan <- iface.AIMessageChunk{Err: ctx.Err()}
				return
			}
		}
	}()

	return streamChan, nil
}

// BindTools implements the ChatModel interface.
func (m *AdvancedMockAnthropicProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	newMock := *m
	newMock.boundTools = make([]tools.Tool, len(toolsToBind))
	copy(newMock.boundTools, toolsToBind)
	return &newMock
}

// GetModelName implements the ChatModel interface.
func (m *AdvancedMockAnthropicProvider) GetModelName() string {
	return m.modelName
}

// GetProviderName implements the LLM interface.
func (m *AdvancedMockAnthropicProvider) GetProviderName() string {
	return ProviderName
}

// Invoke implements the Runnable interface.
func (m *AdvancedMockAnthropicProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	return m.Generate(ctx, messages, options...)
}

// Batch implements the Runnable interface.
func (m *AdvancedMockAnthropicProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := m.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

// Stream implements the Runnable interface.
func (m *AdvancedMockAnthropicProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	chunkChan, err := m.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	outputChan := make(chan any)
	go func() {
		defer close(outputChan)
		for chunk := range chunkChan {
			select {
			case outputChan <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return outputChan, nil
}

// CheckHealth implements the ChatModel interface.
func (m *AdvancedMockAnthropicProvider) CheckHealth() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]any{
		"provider":  ProviderName,
		"model":     m.modelName,
		"status":    "healthy",
		"callCount": m.callCount,
	}
}

// SetError configures the mock to return an error.
func (m *AdvancedMockAnthropicProvider) SetError(shouldError bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorToReturn = err
}

// SetDelay configures the mock to simulate delay.
func (m *AdvancedMockAnthropicProvider) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateDelay = delay
}

// SetRateLimit configures the mock to simulate rate limiting.
func (m *AdvancedMockAnthropicProvider) SetRateLimit(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateRateLimit = enabled
}

// GetCallCount returns the number of times methods have been called.
func (m *AdvancedMockAnthropicProvider) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// Reset resets the mock state.
func (m *AdvancedMockAnthropicProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.shouldError = false
	m.errorToReturn = nil
	m.responseIndex = 0
	m.rateLimitCount = 0
	m.simulateRateLimit = false
	m.simulateDelay = 0
}

// Ensure AdvancedMockAnthropicProvider implements the interfaces.
var (
	_ iface.ChatModel = (*AdvancedMockAnthropicProvider)(nil)
)
