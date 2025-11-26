// Package providers provides concrete implementations of memory interfaces.
// It contains provider-specific implementations that can be swapped out.
package providers

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

// Ensure BaseChatMessageHistory implements the interface.
var _ iface.ChatMessageHistory = (*BaseChatMessageHistory)(nil)

// CompositeChatMessageHistory provides a composable wrapper around other chat message histories.
// It allows chaining multiple histories and adding middleware-like functionality.
type CompositeChatMessageHistory struct {
	primary   iface.ChatMessageHistory
	secondary iface.ChatMessageHistory
	onAddHook func(context.Context, schema.Message) error
	onGetHook func(context.Context, []schema.Message) ([]schema.Message, error)
	maxSize   int
}

// CompositeHistoryOption is a functional option for configuring CompositeChatMessageHistory.
type CompositeHistoryOption func(*CompositeChatMessageHistory)

// NewCompositeChatMessageHistory creates a new composite chat message history.
func NewCompositeChatMessageHistory(primary iface.ChatMessageHistory, options ...CompositeHistoryOption) *CompositeChatMessageHistory {
	h := &CompositeChatMessageHistory{
		primary: primary,
		maxSize: -1, // No limit by default
	}

	for _, option := range options {
		option(h)
	}

	return h
}

// WithSecondaryHistory sets a secondary history for fallback or additional functionality.
func WithSecondaryHistory(secondary iface.ChatMessageHistory) CompositeHistoryOption {
	return func(h *CompositeChatMessageHistory) {
		h.secondary = secondary
	}
}

// WithMaxSize sets the maximum number of messages to keep.
func WithMaxSize(maxSize int) CompositeHistoryOption {
	return func(h *CompositeChatMessageHistory) {
		h.maxSize = maxSize
	}
}

// WithOnAddHook sets a hook function called before adding messages.
func WithOnAddHook(hook func(context.Context, schema.Message) error) CompositeHistoryOption {
	return func(h *CompositeChatMessageHistory) {
		h.onAddHook = hook
	}
}

// WithOnGetHook sets a hook function called after getting messages.
func WithOnGetHook(hook func(context.Context, []schema.Message) ([]schema.Message, error)) CompositeHistoryOption {
	return func(h *CompositeChatMessageHistory) {
		h.onGetHook = hook
	}
}

// AddMessage adds a message to the primary history and optionally to the secondary history.
func (h *CompositeChatMessageHistory) AddMessage(ctx context.Context, message schema.Message) error {
	// Call onAdd hook if provided
	if h.onAddHook != nil {
		if err := h.onAddHook(ctx, message); err != nil {
			return err
		}
	}

	// Add to primary history
	if err := h.primary.AddMessage(ctx, message); err != nil {
		return err
	}

	// Add to secondary history if provided
	if h.secondary != nil {
		if err := h.secondary.AddMessage(ctx, message); err != nil {
			// Log warning but don't fail the operation
			// (In a real implementation, this would use proper logging)
		}
	}

	// Apply size limit if configured
	if h.maxSize > 0 {
		if err := h.applySizeLimit(ctx); err != nil {
			// Log warning but don't fail the operation
			// (In a real implementation, this would use proper logging)
		}
	}

	return nil
}

// AddUserMessage adds a user message through the composite history.
func (h *CompositeChatMessageHistory) AddUserMessage(ctx context.Context, content string) error {
	return h.AddMessage(ctx, schema.NewHumanMessage(content))
}

// AddAIMessage adds an AI message through the composite history.
func (h *CompositeChatMessageHistory) AddAIMessage(ctx context.Context, content string) error {
	return h.AddMessage(ctx, schema.NewAIMessage(content))
}

// GetMessages retrieves messages from the primary history and applies any hooks.
func (h *CompositeChatMessageHistory) GetMessages(ctx context.Context) ([]schema.Message, error) {
	messages, err := h.primary.GetMessages(ctx)
	if err != nil {
		return nil, err
	}

	// Apply onGet hook if provided
	if h.onGetHook != nil {
		messages, err = h.onGetHook(ctx, messages)
		if err != nil {
			return nil, err
		}
	}

	return messages, nil
}

// Clear clears both primary and secondary histories.
func (h *CompositeChatMessageHistory) Clear(ctx context.Context) error {
	// Clear primary history
	if err := h.primary.Clear(ctx); err != nil {
		return err
	}

	// Clear secondary history if provided
	if h.secondary != nil {
		if err := h.secondary.Clear(ctx); err != nil {
			// Log warning but don't fail the operation
			// (In a real implementation, this would use proper logging)
		}
	}

	return nil
}

// applySizeLimit removes oldest messages if the history exceeds the maximum size.
func (h *CompositeChatMessageHistory) applySizeLimit(ctx context.Context) error {
	messages, err := h.primary.GetMessages(ctx)
	if err != nil {
		return err
	}

	if len(messages) <= h.maxSize {
		return nil
	}

	// For BaseChatMessageHistory, we need to recreate it with limited messages
	// This is a simplified implementation - in practice, this would need to be
	// implemented differently based on the underlying storage
	if baseHistory, ok := h.primary.(*BaseChatMessageHistory); ok {
		// Keep only the most recent messages
		baseHistory.mu.Lock()
		baseHistory.messages = messages[len(messages)-h.maxSize:]
		baseHistory.mu.Unlock()
	}

	return nil
}

// Ensure CompositeChatMessageHistory implements the interface.
var _ iface.ChatMessageHistory = (*CompositeChatMessageHistory)(nil)
