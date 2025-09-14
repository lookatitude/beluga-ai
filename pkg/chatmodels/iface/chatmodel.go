// Package iface defines the core interfaces for the chatmodels package.
// It follows the Interface Segregation Principle by providing small, focused interfaces
// that serve specific purposes within the chat model system.
package iface

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MessageGenerator defines the interface for generating messages.
// This focuses solely on message generation capabilities.
type MessageGenerator interface {
	// GenerateMessages takes a list of messages and generates a response.
	// This is the primary method for chat-based interactions.
	GenerateMessages(ctx context.Context, messages []schema.Message, options ...core.Option) ([]schema.Message, error)
}

// StreamMessageHandler defines the interface for streaming message responses.
// This allows for real-time streaming of chat responses.
type StreamMessageHandler interface {
	// StreamMessages provides streaming responses for chat interactions.
	StreamMessages(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan schema.Message, error)
}

// ModelInfoProvider defines the interface for providing model information.
type ModelInfoProvider interface {
	// GetModelInfo returns information about the underlying model.
	GetModelInfo() ModelInfo
}

// HealthChecker defines the interface for health checking chat model components.
type HealthChecker interface {
	// CheckHealth returns the health status information.
	CheckHealth() map[string]interface{}
}

// ChatModel defines the core interface for chat-based language models.
// It combines message generation with model information and health checking capabilities.
// This follows the Interface Segregation Principle while providing a composite interface
// for the most common use cases.
type ChatModel interface {
	MessageGenerator
	StreamMessageHandler
	ModelInfoProvider
	HealthChecker
	core.Runnable // Embed core Runnable for consistency with framework
}

// ModelInfo contains metadata about a chat model.
type ModelInfo struct {
	Name         string
	Provider     string
	Version      string
	MaxTokens    int
	Capabilities []string
}

// Option represents a functional option for configuring chat models.
type Option interface {
	Apply(config *map[string]any)
}

// optionFunc is a helper type that allows an ordinary function to be used as an Option.
type optionFunc func(config *map[string]any)

// Apply calls f(config), allowing optionFunc to satisfy the Option interface.
func (f optionFunc) Apply(config *map[string]any) {
	f(config)
}

// OptionFunc creates a new Option that executes the provided function.
func OptionFunc(f func(config *map[string]any)) Option {
	return optionFunc(f)
}

// Options holds the configuration options for chat models.
type Options struct {
	Temperature     float32
	MaxTokens       int
	TopP            float32
	StopSequences   []string
	SystemPrompt    string
	FunctionCalling bool
	Timeout         time.Duration
	MaxRetries      int
	EnableMetrics   bool
	EnableTracing   bool
}

// ChatModelFactory defines the interface for creating chat model instances.
// It enables dependency injection and different chat model creation strategies.
type ChatModelFactory interface {
	// CreateChatModel creates a new chat model instance based on the provided configuration.
	CreateChatModel(ctx context.Context, config interface{}) (ChatModel, error)
}
