// Package summary provides summary-based memory implementations.
// This file re-exports types from internal/summary for the provider pattern.
package summary

import (
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/summary"
)

// Type aliases for backward compatibility and provider pattern.
type (
	// ConversationSummaryMemory summarizes the conversation history over time.
	ConversationSummaryMemory = summary.ConversationSummaryMemory

	// ConversationSummaryBufferMemory combines buffer memory with summarization.
	ConversationSummaryBufferMemory = summary.ConversationSummaryBufferMemory
)

// NewConversationSummaryMemory creates a new ConversationSummaryMemory.
var NewConversationSummaryMemory = summary.NewConversationSummaryMemory

// NewConversationSummaryBufferMemory creates a new ConversationSummaryBufferMemory.
var NewConversationSummaryBufferMemory = summary.NewConversationSummaryBufferMemory

// DefaultSummaryPrompt is a basic prompt template for summarizing conversations.
var DefaultSummaryPrompt = summary.DefaultSummaryPrompt
