// Package iface defines the core interfaces for Large Language Model interactions.
// These interfaces provide the foundation for implementing chat models and related functionality.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// ChatModel is the interface for chat models.
// It extends the Runnable interface for more complex interactions.
// ChatModel implementations should be thread-safe and support concurrent usage.
type ChatModel interface {
	core.Runnable
	LLM

	// Generate takes a series of messages and returns an AI message.
	// It is a single call to the model with no streaming.
	// The implementation should handle all message types and tool calls appropriately.
	Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error)

	// StreamChat takes a series of messages and returns a channel of AIMessageChunk.
	// This allows for streaming responses from the model.
	// The channel will be closed when the response is complete or an error occurs.
	StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan AIMessageChunk, error)

	// BindTools binds a list of tools to the ChatModel. The returned ChatModel
	// will then be able to call these tools.
	// The specific way tools are bound and used depends on the underlying model provider.
	// Returns a new ChatModel instance with the tools bound to avoid mutating the original.
	BindTools(toolsToBind []core.Tool) ChatModel

	// GetModelName returns the model name used by this ChatModel instance.
	// This is useful for logging, metrics, and debugging.
	GetModelName() string

	// CheckHealth returns the health status information.
	// This allows monitoring the health of chat model providers.
	CheckHealth() map[string]any
}

// AIMessageChunk represents a chunk of an AI message, typically used in streaming.
// It can contain content, tool calls, or an error.
// Chunks should be processed in order and the stream ends when the channel closes.
type AIMessageChunk struct {
	Err            error
	AdditionalArgs map[string]any
	Content        string
	ToolCallChunks []schema.ToolCallChunk
}

// LLM is the interface for basic Large Language Model interactions.
// This provides a simpler interface for text generation without chat history support.
type LLM interface {
	// Invoke sends a single request to the LLM and gets a single response.
	// The input can be any type, but is typically a string prompt. Output is any, typically string.
	Invoke(ctx context.Context, input any, options ...core.Option) (any, error)

	// GetModelName returns the specific model name being used by this LLM instance.
	GetModelName() string

	// GetProviderName returns the name of the LLM provider (e.g., "openai", "anthropic").
	GetProviderName() string
}

// LLMFactory defines the interface for creating LLM instances.
// This allows for different LLM providers to be instantiated based on configuration.
type LLMFactory interface {
	// CreateLLM creates an LLM instance based on the provided configuration.
	CreateLLM(ctx context.Context, config any) (LLM, error)

	// CreateChatModel creates a ChatModel instance based on the provided configuration.
	CreateChatModel(ctx context.Context, config any) (ChatModel, error)
}
