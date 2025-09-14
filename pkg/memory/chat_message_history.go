// Package memory provides interfaces and implementations for managing conversation history.
package memory

import (
	"context"
	"github.com/lookatitude/beluga-ai/schema"
)

// BaseChatMessageHistory implements a simple in-memory message history.
type BaseChatMessageHistory struct {
	messages []schema.Message
}

// NewBaseChatMessageHistory creates a new empty message history.
func NewBaseChatMessageHistory() *BaseChatMessageHistory {
	return &BaseChatMessageHistory{
		messages: make([]schema.Message, 0),
	}
}

// AddMessage adds a generic message to the history.
func (h *BaseChatMessageHistory) AddMessage(ctx context.Context, message schema.Message) error {
	h.messages = append(h.messages, message)
	return nil
}

// AddUserMessage adds a human message to the history.
func (h *BaseChatMessageHistory) AddUserMessage(ctx context.Context, content string) error {
	return h.AddMessage(ctx, schema.NewHumanMessage(content))
}

// AddAIMessage adds an AI message to the history.
func (h *BaseChatMessageHistory) AddAIMessage(ctx context.Context, content string) error {
	return h.AddMessage(ctx, schema.NewAIMessage(content))
}

// GetMessages returns all messages in the history.
func (h *BaseChatMessageHistory) GetMessages(ctx context.Context) ([]schema.Message, error) {
	// Return a copy to prevent modification
	messagesCopy := make([]schema.Message, len(h.messages))
	copy(messagesCopy, h.messages)
	return messagesCopy, nil
}

// Clear removes all messages from the history.
func (h *BaseChatMessageHistory) Clear(ctx context.Context) error {
	h.messages = h.messages[:0] // Efficient way to clear a slice
	return nil
}

// Ensure BaseChatMessageHistory implements the interface.
var _ ChatMessageHistory = (*BaseChatMessageHistory)(nil)

