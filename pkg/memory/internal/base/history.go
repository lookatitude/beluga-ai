// Package base provides base implementations for memory-related types.
// It contains shared code used by various memory providers.
package base

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// BaseChatMessageHistory implements a simple in-memory message history.
type BaseChatMessageHistory struct {
	messages []schema.Message
	maxSize  int // Maximum number of messages to keep (-1 for unlimited)
	mu       sync.RWMutex
}

// BaseHistoryOption is a functional option for configuring BaseChatMessageHistory.
type BaseHistoryOption func(*BaseChatMessageHistory)

// NewBaseChatMessageHistory creates a new empty message history with functional options.
func NewBaseChatMessageHistory(options ...BaseHistoryOption) *BaseChatMessageHistory {
	h := &BaseChatMessageHistory{
		messages: make([]schema.Message, 0),
		maxSize:  -1, // Unlimited by default
	}

	for _, option := range options {
		option(h)
	}

	return h
}

// WithMaxHistorySize sets the maximum number of messages to keep in the history.
func WithMaxHistorySize(maxSize int) BaseHistoryOption {
	return func(h *BaseChatMessageHistory) {
		h.maxSize = maxSize
	}
}

// AddMessage adds a generic message to the history.
func (h *BaseChatMessageHistory) AddMessage(ctx context.Context, message schema.Message) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, message)

	// Apply size limit if configured
	if h.maxSize > 0 && len(h.messages) > h.maxSize {
		// Remove oldest messages to maintain the size limit
		h.messages = h.messages[len(h.messages)-h.maxSize:]
	}

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
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	// Return a copy to prevent modification
	messagesCopy := make([]schema.Message, len(h.messages))
	copy(messagesCopy, h.messages)
	return messagesCopy, nil
}

// Clear removes all messages from the history.
func (h *BaseChatMessageHistory) Clear(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = h.messages[:0] // Efficient way to clear a slice
	return nil
}

// GetMaxSize returns the current max size setting (for testing).
func (h *BaseChatMessageHistory) GetMaxSize() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.maxSize
}

// Ensure BaseChatMessageHistory implements the interface.
var _ iface.ChatMessageHistory = (*BaseChatMessageHistory)(nil)
