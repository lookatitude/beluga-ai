// Package buffer provides buffer memory implementations.
// This file re-exports types from internal/buffer for the provider pattern.
package buffer

import (
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/buffer"
)

// Type aliases for backward compatibility and provider pattern.
type (
	// ChatMessageBufferMemory is a simple memory implementation that stores all messages in a buffer.
	ChatMessageBufferMemory = buffer.ChatMessageBufferMemory
)

// NewChatMessageBufferMemory creates a new buffer memory with default settings.
var NewChatMessageBufferMemory = buffer.NewChatMessageBufferMemory
