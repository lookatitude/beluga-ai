package providers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockBaseChatMessageHistory provides a comprehensive mock implementation for testing BaseChatMessageHistory.
type AdvancedMockBaseChatMessageHistory struct {
	errorToReturn error
	mock.Mock
	messages          []schema.Message
	callCount         int
	simulateDelay     time.Duration
	rateLimitCount    int
	maxSize           int
	mu                sync.RWMutex
	shouldError       bool
	simulateRateLimit bool
}

// NewAdvancedMockBaseChatMessageHistory creates a new advanced mock with configurable behavior.
func NewAdvancedMockBaseChatMessageHistory() *AdvancedMockBaseChatMessageHistory {
	mock := &AdvancedMockBaseChatMessageHistory{
		messages: make([]schema.Message, 0),
		maxSize:  -1, // Unlimited by default
	}
	return mock
}

// AddMessage implements the ChatMessageHistory interface.
func (m *AdvancedMockBaseChatMessageHistory) AddMessage(ctx context.Context, message schema.Message) error {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return errorToReturn
		}
		return errors.New("mock error")
	}

	m.mu.Lock()
	m.messages = append(m.messages, message)

	// Apply size limit if configured
	if m.maxSize > 0 && len(m.messages) > m.maxSize {
		m.messages = m.messages[len(m.messages)-m.maxSize:]
	}
	m.mu.Unlock()

	return nil
}

// AddUserMessage implements the ChatMessageHistory interface.
func (m *AdvancedMockBaseChatMessageHistory) AddUserMessage(ctx context.Context, content string) error {
	return m.AddMessage(ctx, schema.NewHumanMessage(content))
}

// AddAIMessage implements the ChatMessageHistory interface.
func (m *AdvancedMockBaseChatMessageHistory) AddAIMessage(ctx context.Context, content string) error {
	return m.AddMessage(ctx, schema.NewAIMessage(content))
}

// GetMessages implements the ChatMessageHistory interface.
func (m *AdvancedMockBaseChatMessageHistory) GetMessages(ctx context.Context) ([]schema.Message, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, errors.New("mock error")
	}

	m.mu.RLock()
	messagesCopy := make([]schema.Message, len(m.messages))
	copy(messagesCopy, m.messages)
	m.mu.RUnlock()

	return messagesCopy, nil
}

// Clear implements the ChatMessageHistory interface.
func (m *AdvancedMockBaseChatMessageHistory) Clear(ctx context.Context) error {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return errorToReturn
		}
		return errors.New("mock error")
	}

	m.mu.Lock()
	m.messages = m.messages[:0]
	m.mu.Unlock()

	return nil
}

// SetError configures the mock to return an error.
func (m *AdvancedMockBaseChatMessageHistory) SetError(shouldError bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorToReturn = err
}

// SetDelay configures the mock to simulate delay.
func (m *AdvancedMockBaseChatMessageHistory) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateDelay = delay
}

// SetMaxSize configures the mock maximum size.
func (m *AdvancedMockBaseChatMessageHistory) SetMaxSize(maxSize int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxSize = maxSize
}

// SetRateLimit configures the mock to simulate rate limiting.
func (m *AdvancedMockBaseChatMessageHistory) SetRateLimit(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateRateLimit = enabled
}

// GetCallCount returns the number of times methods have been called.
func (m *AdvancedMockBaseChatMessageHistory) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetMessagesCount returns the current number of messages stored.
func (m *AdvancedMockBaseChatMessageHistory) GetMessagesCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

// Reset resets the mock state.
func (m *AdvancedMockBaseChatMessageHistory) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.shouldError = false
	m.errorToReturn = nil
	m.messages = m.messages[:0]
	m.rateLimitCount = 0
	m.simulateRateLimit = false
	m.simulateDelay = 0
}

// Ensure AdvancedMockBaseChatMessageHistory implements the interface.
var (
	_ iface.ChatMessageHistory = (*AdvancedMockBaseChatMessageHistory)(nil)
)
