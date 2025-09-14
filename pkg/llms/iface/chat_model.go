// Package iface defines the core interfaces for Large Language Model interactions.
// These interfaces provide the foundation for implementing chat models and related functionality.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// ChatModel is the interface for chat models.
// It extends the Runnable interface for more complex interactions.
// ChatModel implementations should be thread-safe and support concurrent usage.
type ChatModel interface {
	core.Runnable

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
	BindTools(toolsToBind []tools.Tool) ChatModel

	// GetModelName returns the model name used by this ChatModel instance.
	// This is useful for logging, metrics, and debugging.
	GetModelName() string
}

// AIMessageChunk represents a chunk of an AI message, typically used in streaming.
// It can contain content, tool calls, or an error.
// Chunks should be processed in order and the stream ends when the channel closes.
type AIMessageChunk struct {
	Content        string                 // Text content of the chunk
	ToolCallChunks []schema.ToolCallChunk // Tool call information if present
	AdditionalArgs map[string]interface{} // Provider-specific arguments or metadata
	Err            error                  // Error encountered during streaming for this chunk
}

// LLM is the interface for basic Large Language Model interactions.
// This provides a simpler interface for text generation without chat history support.
type LLM interface {
	// Invoke sends a single request to the LLM and gets a single response.
	// The input prompt can be a simple string.
	Invoke(ctx context.Context, prompt string, options ...core.Option) (string, error)

	// GetModelName returns the specific model name being used by this LLM instance.
	GetModelName() string

	// GetProviderName returns the name of the LLM provider (e.g., "openai", "anthropic").
	GetProviderName() string
}

// LLMFactory defines the interface for creating LLM instances.
// This allows for different LLM providers to be instantiated based on configuration.
type LLMFactory interface {
	// CreateLLM creates an LLM instance based on the provided configuration.
	CreateLLM(ctx context.Context, config interface{}) (LLM, error)

	// CreateChatModel creates a ChatModel instance based on the provided configuration.
	CreateChatModel(ctx context.Context, config interface{}) (ChatModel, error)
}
