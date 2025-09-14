// Package mock provides a mock implementation of the llms.ChatModel interface
// for testing and development purposes.
package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/llms/internal/common"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Provider constants
const (
	ProviderName = "mock"
	DefaultModel = "mock-model"
)

// MockProvider implements the ChatModel interface for testing and development
type MockProvider struct {
	config      *llms.Config
	modelName   string
	responses   []string
	toolResults map[string]string
	tools       []tools.Tool
	metrics     llms.MetricsRecorder
	tracing     *common.TracingHelper
	callCount   int
	shouldError bool
}

// NewMockProvider creates a new mock provider instance
func NewMockProvider(config *llms.Config) (*MockProvider, error) {
	// Set default model if not specified
	modelName := config.ModelName
	if modelName == "" {
		modelName = DefaultModel
	}

	// Get responses from provider-specific config
	var responses []string
	if respSlice, ok := config.ProviderSpecific["responses"].([]interface{}); ok {
		for _, r := range respSlice {
			if str, ok := r.(string); ok {
				responses = append(responses, str)
			}
		}
	}

	// Default responses if none provided
	if len(responses) == 0 {
		responses = []string{
			"This is a mock response from the AI assistant.",
			"I understand your request and I'm here to help.",
			"Thank you for your question. Here's my response.",
		}
	}

	provider := &MockProvider{
		config:      config,
		modelName:   modelName,
		responses:   responses,
		toolResults: make(map[string]string),
		metrics:     llms.GetMetrics(), // Get global metrics instance
		tracing:     common.NewTracingHelper(),
		callCount:   0,
		shouldError: false,
	}

	// Check if should error from config
	if shouldErr, ok := config.ProviderSpecific["should_error"].(bool); ok {
		provider.shouldError = shouldErr
	}

	return provider, nil
}

// Generate implements the ChatModel interface
func (m *MockProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	// Start tracing
	ctx = m.tracing.StartOperation(ctx, "mock.generate", ProviderName, m.modelName)
	defer m.tracing.EndSpan(ctx)

	// Record request metrics
	m.metrics.IncrementActiveRequests(ctx, ProviderName, m.modelName)
	defer m.metrics.DecrementActiveRequests(ctx, ProviderName, m.modelName)

	m.callCount++

	// Check if should error
	if m.shouldError {
		err := llms.NewLLMError("mock.generate", llms.ErrCodeInternalError, fmt.Errorf("mock provider configured to error"))
		m.metrics.RecordError(ctx, ProviderName, m.modelName, llms.ErrCodeInternalError)
		m.tracing.RecordError(ctx, err)
		return nil, err
	}

	// Simulate processing delay
	time.Sleep(10 * time.Millisecond)

	// Get response based on call count
	responseIndex := (m.callCount - 1) % len(m.responses)
	response := m.responses[responseIndex]

	// Create AI message
	aiMsg := schema.NewAIMessage(response)

	// Add mock usage information
	args := aiMsg.AdditionalArgs()
	args["usage"] = map[string]int{
		"input_tokens":  len(messages) * 10, // Mock token count
		"output_tokens": len(response) / 4,  // Mock token count
		"total_tokens":  len(messages)*10 + len(response)/4,
	}

	// Record success metrics
	m.metrics.RecordRequest(ctx, ProviderName, m.modelName, 10*time.Millisecond)

	return aiMsg, nil
}

// StreamChat implements the ChatModel interface
func (m *MockProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	// Start tracing
	ctx = m.tracing.StartOperation(ctx, "mock.stream", ProviderName, m.modelName)
	defer m.tracing.EndSpan(ctx)

	// Record request metrics
	m.metrics.IncrementActiveRequests(ctx, ProviderName, m.modelName)
	defer m.metrics.DecrementActiveRequests(ctx, ProviderName, m.modelName)

	m.callCount++

	// Check if should error
	if m.shouldError {
		err := llms.NewLLMError("mock.stream", llms.ErrCodeInternalError, fmt.Errorf("mock provider configured to error"))
		m.metrics.RecordError(ctx, ProviderName, m.modelName, llms.ErrCodeInternalError)
		m.tracing.RecordError(ctx, err)
		return nil, err
	}

	outputChan := make(chan iface.AIMessageChunk)

	go func() {
		defer close(outputChan)

		// Get response based on call count
		responseIndex := (m.callCount - 1) % len(m.responses)
		response := m.responses[responseIndex]

		// Simulate streaming by sending chunks
		chunkSize := 5
		for i := 0; i < len(response); i += chunkSize {
			select {
			case <-ctx.Done():
				return
			default:
				end := i + chunkSize
				if end > len(response) {
					end = len(response)
				}

				chunk := iface.AIMessageChunk{
					Content:        response[i:end],
					AdditionalArgs: make(map[string]interface{}),
				}

				// Add finish reason on last chunk
				if end == len(response) {
					chunk.AdditionalArgs["finish_reason"] = "stop"
				}

				select {
				case outputChan <- chunk:
				case <-ctx.Done():
					return
				}

				// Simulate typing delay
				time.Sleep(5 * time.Millisecond)
			}
		}

		// Send final chunk with usage information
		finalChunk := iface.AIMessageChunk{
			AdditionalArgs: map[string]interface{}{
				"usage": map[string]int{
					"input_tokens":  len(messages) * 10,
					"output_tokens": len(response) / 4,
					"total_tokens":  len(messages)*10 + len(response)/4,
				},
			},
		}

		select {
		case outputChan <- finalChunk:
		case <-ctx.Done():
		}
	}()

	// Record success metrics
	m.metrics.RecordRequest(ctx, ProviderName, m.modelName, 0)

	return outputChan, nil
}

// BindTools implements the ChatModel interface
func (m *MockProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	newProvider := *m // Create a copy
	newProvider.tools = make([]tools.Tool, len(toolsToBind))
	copy(newProvider.tools, toolsToBind)

	// Set up mock tool results
	newProvider.toolResults = make(map[string]string)
	for _, tool := range toolsToBind {
		newProvider.toolResults[tool.Name()] = fmt.Sprintf("Mock result from %s tool", tool.Name())
	}

	return &newProvider
}

// GetModelName implements the ChatModel interface
func (m *MockProvider) GetModelName() string {
	return m.modelName
}

// Invoke implements the Runnable interface
func (m *MockProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	return m.Generate(ctx, messages, options...)
}

// Batch implements the Runnable interface
func (m *MockProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))

	for i, input := range inputs {
		select {
		case <-ctx.Done():
			return results[:i], ctx.Err()
		default:
			result, err := m.Invoke(ctx, input, options...)
			if err != nil {
				return results[:i], err
			}
			results[i] = result
		}
	}

	return results, nil
}

// Stream implements the Runnable interface
func (m *MockProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	chunkChan, err := m.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	// Convert AIMessageChunk channel to any channel
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

// GetCallCount returns the number of calls made to this provider
func (m *MockProvider) GetCallCount() int {
	return m.callCount
}

// SetResponses sets the mock responses
func (m *MockProvider) SetResponses(responses []string) {
	m.responses = responses
}

// SetShouldError sets whether the provider should return errors
func (m *MockProvider) SetShouldError(shouldError bool) {
	m.shouldError = shouldError
}

// Reset resets the provider state
func (m *MockProvider) Reset() {
	m.callCount = 0
	m.toolResults = make(map[string]string)
}

// AddToolResult adds a mock result for a specific tool
func (m *MockProvider) AddToolResult(toolName, result string) {
	if m.toolResults == nil {
		m.toolResults = make(map[string]string)
	}
	m.toolResults[toolName] = result
}

// Factory function for creating mock providers
func NewMockProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
	return func(config *llms.Config) (iface.ChatModel, error) {
		return NewMockProvider(config)
	}
}
