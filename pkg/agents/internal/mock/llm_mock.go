// Package mock provides mocks for agents internal testing.
// This file provides LLM streaming mocks specifically for agent testing.
package mock

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockLLMStreamChat provides a mock implementation of LLM StreamChat for agent testing.
type MockLLMStreamChat struct {
	responses      []string
	streamingDelay time.Duration
	shouldError    bool
	errorToReturn  error
	callCount      int
	mu             sync.RWMutex
	toolCallChunks []schema.ToolCallChunk
	simulateDelay  bool
}

// NewMockLLMStreamChat creates a new mock LLM StreamChat.
func NewMockLLMStreamChat() *MockLLMStreamChat {
	return &MockLLMStreamChat{
		responses:      []string{"Default mock response"},
		streamingDelay: 10 * time.Millisecond,
		shouldError:    false,
		toolCallChunks: make([]schema.ToolCallChunk, 0),
	}
}

// WithResponses sets the responses to return.
func (m *MockLLMStreamChat) WithResponses(responses ...string) *MockLLMStreamChat {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = responses
	return m
}

// WithStreamingDelay sets the delay between chunks.
func (m *MockLLMStreamChat) WithStreamingDelay(delay time.Duration) *MockLLMStreamChat {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamingDelay = delay
	return m
}

// WithError configures the mock to return an error.
func (m *MockLLMStreamChat) WithError(err error) *MockLLMStreamChat {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = true
	m.errorToReturn = err
	return m
}

// WithToolCallChunks sets tool call chunks to include in responses.
func (m *MockLLMStreamChat) WithToolCallChunks(chunks []schema.ToolCallChunk) *MockLLMStreamChat {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolCallChunks = chunks
	return m
}

// StreamChat implements the ChatModel StreamChat interface.
func (m *MockLLMStreamChat) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	response := m.responses[m.callCount%len(m.responses)]
	streamingDelay := m.streamingDelay
	simulateDelay := m.simulateDelay
	toolCallChunks := m.toolCallChunks
	m.mu.Unlock()

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, fmt.Errorf("mock LLM error")
	}

	ch := make(chan iface.AIMessageChunk, 10)

	go func() {
		defer close(ch)

		words := strings.Fields(response)

		for i, word := range words {
			if simulateDelay && streamingDelay > 0 {
				select {
				case <-ctx.Done():
					ch <- iface.AIMessageChunk{Err: ctx.Err()}
					return
				case <-time.After(streamingDelay):
				}
			}

			chunk := iface.AIMessageChunk{
				Content: word + " ",
			}

			select {
			case <-ctx.Done():
				ch <- iface.AIMessageChunk{Err: ctx.Err()}
				return
			case ch <- chunk:
			}

			if i == len(words)-1 && len(toolCallChunks) > 0 {
				chunk := iface.AIMessageChunk{
					ToolCallChunks: toolCallChunks,
				}
				select {
				case <-ctx.Done():
					ch <- iface.AIMessageChunk{Err: ctx.Err()}
					return
				case ch <- chunk:
				}
			}
		}
	}()

	return ch, nil
}

// GetCallCount returns the number of StreamChat calls.
func (m *MockLLMStreamChat) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}
