// Package mock provides mock implementations for testing memory components.
// This file re-exports types from internal/mock for the provider pattern.
package mock

import (
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/mock"
)

// Type aliases for backward compatibility and provider pattern.
type (
	// MockChatMessageHistory is a mock implementation of ChatMessageHistory for testing.
	MockChatMessageHistory = mock.MockChatMessageHistory
)

// NewMockChatMessageHistory creates a new mock chat message history.
var NewMockChatMessageHistory = mock.NewMockChatMessageHistory
