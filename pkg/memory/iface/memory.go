// Package iface defines the core interfaces for the memory package.
// It follows the Interface Segregation Principle by providing focused interfaces
// for different aspects of memory management.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Memory defines the interface for all memory implementations.
// It provides methods for managing conversation history and context variables.
type Memory interface {
	// MemoryVariables returns the list of variable names that the memory makes available.
	MemoryVariables() []string

	// LoadMemoryVariables loads memory variables given the context and input values.
	LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error)

	// SaveContext saves the current context and new inputs/outputs to memory.
	SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error

	// Clear clears the memory contents.
	Clear(ctx context.Context) error
}

// ChatMessageHistory defines the interface for storing and retrieving message history.
// It provides methods for managing a sequence of chat messages.
type ChatMessageHistory interface {
	// AddMessage adds a message to the history.
	AddMessage(ctx context.Context, message schema.Message) error

	// AddUserMessage adds a human message to the history.
	AddUserMessage(ctx context.Context, content string) error

	// AddAIMessage adds an AI message to the history.
	AddAIMessage(ctx context.Context, content string) error

	// GetMessages returns all messages in the history.
	GetMessages(ctx context.Context) ([]schema.Message, error)

	// Clear removes all messages from the history.
	Clear(ctx context.Context) error
}
