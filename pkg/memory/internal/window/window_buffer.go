// Package window provides window-based memory implementations.
// It contains implementations that maintain a fixed-size window of recent interactions.
package window

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/buffer"
)

// ConversationWindowBufferMemory keeps only the last K interactions (2K messages).
type ConversationWindowBufferMemory struct {
	*buffer.ChatMessageBufferMemory
	K int // Number of conversation turns to keep
}

// NewConversationWindowBufferMemory creates a new window buffer memory.
func NewConversationWindowBufferMemory(history iface.ChatMessageHistory, k int) *ConversationWindowBufferMemory {
	if k <= 0 {
		k = 5 // Default to last 5 interactions if invalid k
	}
	return &ConversationWindowBufferMemory{
		ChatMessageBufferMemory: buffer.NewChatMessageBufferMemory(history),
		K:                       k,
	}
}

// LoadMemoryVariables loads and returns only the last K interactions.
func (m *ConversationWindowBufferMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	messages, err := m.ChatHistory.GetMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages from chat history: %w", err)
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
	bufferStr := getBufferString(messages, m.HumanPrefix, m.AIPrefix)
	return map[string]any{m.MemoryKey: bufferStr}, nil
}

// Ensure ConversationWindowBufferMemory implements the Memory interface.
var _ iface.Memory = (*ConversationWindowBufferMemory)(nil)
