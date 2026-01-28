// Package window provides window-based memory implementations.
// This file re-exports types from internal/window for the provider pattern.
package window

import (
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/window"
)

// Type aliases for backward compatibility and provider pattern.
type (
	// ConversationBufferWindowMemory remembers a fixed number of the most recent interactions.
	ConversationBufferWindowMemory = window.ConversationBufferWindowMemory
)

// NewConversationBufferWindowMemory creates a new ConversationBufferWindowMemory.
var NewConversationBufferWindowMemory = window.NewConversationBufferWindowMemory
