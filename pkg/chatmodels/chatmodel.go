// Package chatmodels defines interfaces for chat-based language model integrations.
// This package extends the base LLM functionality with chat-specific features.
package chatmodels

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// ChatModel represents a chat-based language model that can handle conversation-like interactions.
type ChatModel interface {
	core.Runnable

	// GenerateMessages takes a list of messages and generates a response.
	// This is the primary method for chat-based interactions.
	GenerateMessages(ctx context.Context, messages []schema.Message, options ...core.Option) ([]schema.Message, error)

	// StreamMessages provides streaming responses for chat interactions.
	StreamMessages(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan schema.Message, error)

	// GetModelInfo returns information about the underlying model.
	GetModelInfo() ModelInfo
}

// ModelInfo contains metadata about a chat model.
type ModelInfo struct {
	Name         string
	Provider     string
	Version      string
	MaxTokens    int
	Capabilities []string
}

// ChatModelOption represents options that can be applied to chat model operations.
type ChatModelOption func(*ChatModelConfig)

// ChatModelConfig holds configuration for chat model operations.
type ChatModelConfig struct {
	Temperature     float32
	MaxTokens       int
	TopP           float32
	StopSequences   []string
	SystemPrompt    string
	FunctionCalling bool
}

// WithTemperature sets the temperature for response generation.
func WithTemperature(temp float32) ChatModelOption {
	return func(c *ChatModelConfig) {
		c.Temperature = temp
	}
}

// WithMaxTokens sets the maximum number of tokens to generate.
func WithMaxTokens(maxTokens int) ChatModelOption {
	return func(c *ChatModelConfig) {
		c.MaxTokens = maxTokens
	}
}
