// Package memory provides interfaces and implementations for managing conversation history.
package memory

import (
	"context"
)

// ConversationWindowBufferMemory keeps only the last K interactions (2K messages).
type ConversationWindowBufferMemory struct {
	*ChatMessageBufferMemory
	K int // Number of conversation turns to keep
}

// NewConversationWindowBufferMemory creates a new window buffer memory.
func NewConversationWindowBufferMemory(history ChatMessageHistory, k int) *ConversationWindowBufferMemory {
	if k <= 0 {
		k = 5 // Default to last 5 interactions if invalid k
	}
	return &ConversationWindowBufferMemory{
		ChatMessageBufferMemory: NewChatMessageBufferMemory(history),
		K:                       k,
	}
}

// LoadMemoryVariables loads and returns only the last K interactions.
func (m *ConversationWindowBufferMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	messages, err := m.ChatHistory.GetMessages(ctx)
	if err != nil {
		return nil, err
	}

	// Apply window to keep only the last K pairs (or fewer if not enough messages)
	// Each pair is user + AI message, so 2*K messages total
	windowSize := 2 * m.K
	if len(messages) > windowSize {
		messages = messages[len(messages)-windowSize:]
	}

	if m.ReturnMessages {
		return map[string]any{m.MemoryKey: messages}, nil
	}

	// Format messages as a string
	buffer := GetBufferString(messages, m.HumanPrefix, m.AIPrefix)
	return map[string]any{m.MemoryKey: buffer}, nil
}

// Ensure ConversationWindowBufferMemory implements the Memory interface.
var _ Memory = (*ConversationWindowBufferMemory)(nil)

