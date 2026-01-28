// Package providers provides concrete implementations of memory interfaces.
// It contains provider-specific implementations that can be swapped out.
//
// This file re-exports base history types from internal/base for backward compatibility.
package providers

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/memory/internal/base"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Type aliases for backward compatibility.
type (
	// BaseChatMessageHistory implements a simple in-memory message history.
	BaseChatMessageHistory = base.BaseChatMessageHistory

	// BaseHistoryOption is a functional option for configuring BaseChatMessageHistory.
	BaseHistoryOption = base.BaseHistoryOption

	// CompositeChatMessageHistory provides a composable wrapper around other chat message histories.
	CompositeChatMessageHistory = base.CompositeChatMessageHistory

	// CompositeHistoryOption is a functional option for configuring CompositeChatMessageHistory.
	CompositeHistoryOption = base.CompositeHistoryOption

	// AdvancedMockBaseChatMessageHistory provides a comprehensive mock implementation for testing.
	AdvancedMockBaseChatMessageHistory = base.AdvancedMockBaseChatMessageHistory
)

// NewBaseChatMessageHistory creates a new empty message history with functional options.
var NewBaseChatMessageHistory = base.NewBaseChatMessageHistory

// WithMaxHistorySize sets the maximum number of messages to keep in the history.
var WithMaxHistorySize = base.WithMaxHistorySize

// NewCompositeChatMessageHistory creates a new composite chat message history.
var NewCompositeChatMessageHistory = base.NewCompositeChatMessageHistory

// WithSecondaryHistory sets a secondary history for fallback or additional functionality.
var WithSecondaryHistory = base.WithSecondaryHistory

// WithMaxSize sets the maximum number of messages to keep.
var WithMaxSize = base.WithMaxSize

// WithOnAddHook sets a hook function called before adding messages.
func WithOnAddHook(hook func(context.Context, schema.Message) error) CompositeHistoryOption {
	return base.WithOnAddHook(hook)
}

// WithOnGetHook sets a hook function called after getting messages.
func WithOnGetHook(hook func(context.Context, []schema.Message) ([]schema.Message, error)) CompositeHistoryOption {
	return base.WithOnGetHook(hook)
}

// NewAdvancedMockBaseChatMessageHistory creates a new advanced mock with configurable behavior.
var NewAdvancedMockBaseChatMessageHistory = base.NewAdvancedMockBaseChatMessageHistory
