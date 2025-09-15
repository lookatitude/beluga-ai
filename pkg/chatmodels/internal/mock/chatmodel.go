package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// mockMetrics is a simple stub to avoid import cycles
type mockMetrics struct{}

func (m *mockMetrics) RecordMessageGeneration(model, provider string, duration time.Duration, success bool, tokenCount int) {
	// No-op implementation
}

func (m *mockMetrics) RecordStreamingSession(model, provider string, duration time.Duration, success bool, messageCount int) {
	// No-op implementation
}

// MockChatModel is a mock implementation of the ChatModel interface for testing.
type MockChatModel struct {
	model     string
	config    interface{} // Use interface{} to avoid import cycle
	options   *iface.Options
	metrics   interface{} // Use interface{} to avoid import cycle
	responses []string    // Predefined responses to cycle through
	index     int         // Current response index
}

// NewMockChatModel creates a new mock chat model instance.
func NewMockChatModel(model string, config interface{}, options *iface.Options) (*MockChatModel, error) {
	var metrics interface{}
	// Simple metrics stub to avoid import cycle
	metrics = &mockMetrics{}

	return &MockChatModel{
		model:   model,
		config:  config,
		options: options,
		metrics: metrics,
		responses: []string{
			"This is a mock response from the chat model.",
			"I'm a simulated AI assistant responding to your message.",
			"Thank you for your input. This is a test response.",
			"As a mock model, I'm providing a predefined answer.",
			"Your message has been processed by the mock chat model.",
		},
		index: 0,
	}, nil
}

// GenerateMessages generates mock response messages.
func (m *MockChatModel) GenerateMessages(ctx context.Context, messages []schema.Message, opts ...core.Option) ([]schema.Message, error) {
	start := time.Now()

	// Apply options if provided
	configMap := make(map[string]any)
	for _, opt := range opts {
		opt.Apply(&configMap)
	}

	// Simulate processing time
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond): // Mock delay
	}

	// Get next response in cycle
	response := m.getNextResponse()

	// Create response message
	responseMessage := schema.NewAIMessage(response)

	result := []schema.Message{responseMessage}

	// Record metrics
	duration := time.Since(start)
	if metrics, ok := m.metrics.(*mockMetrics); ok {
		metrics.RecordMessageGeneration(m.model, "mock", duration, true, len(response))
	}

	return result, nil
}

// StreamMessages provides streaming mock responses.
func (m *MockChatModel) StreamMessages(ctx context.Context, messages []schema.Message, opts ...core.Option) (<-chan schema.Message, error) {
	messageChan := make(chan schema.Message, 10)

	go func() {
		defer close(messageChan)

		start := time.Now()
		messageCount := 0

		// Apply options if provided
		configMap := make(map[string]any)
		for _, opt := range opts {
			opt.Apply(&configMap)
		}

		// Simulate streaming by sending chunks
		response := m.getNextResponse()
		words := splitIntoWords(response)

		for i, word := range words {
			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Millisecond): // Mock delay between chunks
				message := schema.NewAIMessage(word)

				// Mark as final chunk
				if i == len(words)-1 {
					// For final chunk, we can use the message as-is since it's the last one
					// The finished flag is implicit in the channel closing
				}

				select {
				case messageChan <- message:
					messageCount++
				case <-ctx.Done():
					return
				}
			}
		}

		// Record metrics
		duration := time.Since(start)
		if metrics, ok := m.metrics.(*mockMetrics); ok {
			metrics.RecordStreamingSession(m.model, "mock", duration, true, messageCount)
		}
	}()

	return messageChan, nil
}

// GetModelInfo returns information about the mock model.
func (m *MockChatModel) GetModelInfo() iface.ModelInfo {
	return iface.ModelInfo{
		Name:      m.model,
		Provider:  "mock",
		Version:   "1.0.0",
		MaxTokens: 4096,
		Capabilities: []string{
			"text-generation",
			"streaming",
			"mock-responses",
		},
	}
}

// CheckHealth returns the health status of the mock model.
func (m *MockChatModel) CheckHealth() map[string]interface{} {
	return map[string]interface{}{
		"status":           "healthy",
		"model":            m.model,
		"provider":         "mock",
		"last_check":       time.Now().Format(time.RFC3339),
		"response_time_ms": 100,
	}
}

// Invoke implements the core.Runnable interface.
func (m *MockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	// Convert input to messages
	messages, ok := input.([]schema.Message)
	if !ok {
		return nil, fmt.Errorf("input must be []schema.Message, got %T", input)
	}

	result, err := m.GenerateMessages(ctx, messages)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Batch implements the core.Runnable interface.
func (m *MockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
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

// Stream implements the core.Runnable interface.
func (m *MockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	// Convert input to messages
	messages, ok := input.([]schema.Message)
	if !ok {
		return nil, fmt.Errorf("input must be []schema.Message, got %T", input)
	}

	messageChan, err := m.StreamMessages(ctx, messages)
	if err != nil {
		return nil, err
	}

	// Convert message channel to any channel
	anyChan := make(chan any, 10)
	go func() {
		defer close(anyChan)
		for msg := range messageChan {
			anyChan <- msg
		}
	}()

	return anyChan, nil
}

// Run implements the core.Runnable interface.
func (m *MockChatModel) Run(ctx context.Context) error {
	// Mock models don't need to run anything continuously
	<-ctx.Done()
	return ctx.Err()
}

// getNextResponse returns the next response in the cycle.
func (m *MockChatModel) getNextResponse() string {
	response := m.responses[m.index]
	m.index = (m.index + 1) % len(m.responses)
	return response
}

// splitIntoWords splits a string into words for streaming simulation.
func splitIntoWords(text string) []string {
	// Simple word splitting - in a real implementation, this would be more sophisticated
	var words []string
	current := ""
	for _, char := range text {
		if char == ' ' || char == '\n' || char == '\t' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
			if char == '\n' {
				words = append(words, "\n")
			} else {
				words = append(words, " ")
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}
