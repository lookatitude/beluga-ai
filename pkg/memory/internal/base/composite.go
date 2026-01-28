// Package base provides base implementations for memory-related types.
package base

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

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
			_ = err
		}
	}

	// Apply size limit if configured
	if h.maxSize > 0 {
		if err := h.applySizeLimit(ctx); err != nil {
			// Log warning but don't fail the operation
			// (In a real implementation, this would use proper logging)
			_ = err
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
			_ = err
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

// GetSecondary returns the secondary history (for testing).
func (h *CompositeChatMessageHistory) GetSecondary() iface.ChatMessageHistory {
	return h.secondary
}

// GetMaxSize returns the max size setting (for testing).
func (h *CompositeChatMessageHistory) GetMaxSize() int {
	return h.maxSize
}

// Ensure CompositeChatMessageHistory implements the interface.
var _ iface.ChatMessageHistory = (*CompositeChatMessageHistory)(nil)
