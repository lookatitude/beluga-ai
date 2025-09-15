// Package mock provides mock implementations for testing memory components.
package mock

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockChatMessageHistory is a mock implementation of ChatMessageHistory for testing.
type MockChatMessageHistory struct {
	MockAddMessage      func(ctx context.Context, message schema.Message) error
	MockAddUserMessage  func(ctx context.Context, content string) error
	MockAddAIMessage    func(ctx context.Context, content string) error
	MockGetMessages     func(ctx context.Context) ([]schema.Message, error)
	MockClear           func(ctx context.Context) error
	messages            []schema.Message
}

// NewMockChatMessageHistory creates a new mock chat message history.
func NewMockChatMessageHistory() *MockChatMessageHistory {
	return &MockChatMessageHistory{
		messages: make([]schema.Message, 0),
	}
}

// AddMessage adds a message to the mock history.
func (m *MockChatMessageHistory) AddMessage(ctx context.Context, message schema.Message) error {
	if m.MockAddMessage != nil {
		return m.MockAddMessage(ctx, message)
	}
	m.messages = append(m.messages, message)
	return nil
}

// AddUserMessage adds a user message to the mock history.
func (m *MockChatMessageHistory) AddUserMessage(ctx context.Context, content string) error {
	if m.MockAddUserMessage != nil {
		return m.MockAddUserMessage(ctx, content)
	}
	return m.AddMessage(ctx, schema.NewHumanMessage(content))
}

// AddAIMessage adds an AI message to the mock history.
func (m *MockChatMessageHistory) AddAIMessage(ctx context.Context, content string) error {
	if m.MockAddAIMessage != nil {
		return m.MockAddAIMessage(ctx, content)
	}
	return m.AddMessage(ctx, schema.NewAIMessage(content))
}

// GetMessages retrieves messages from the mock history.
func (m *MockChatMessageHistory) GetMessages(ctx context.Context) ([]schema.Message, error) {
	if m.MockGetMessages != nil {
		return m.MockGetMessages(ctx)
	}
	// Return a copy to prevent modification
	messagesCopy := make([]schema.Message, len(m.messages))
	copy(messagesCopy, m.messages)
	return messagesCopy, nil
}

// Clear clears the mock history.
func (m *MockChatMessageHistory) Clear(ctx context.Context) error {
	if m.MockClear != nil {
		return m.MockClear(ctx)
	}
	m.messages = m.messages[:0]
	return nil
}

// Ensure MockChatMessageHistory implements the interface.
var _ iface.ChatMessageHistory = (*MockChatMessageHistory)(nil)
