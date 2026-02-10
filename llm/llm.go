package llm

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/schema"
)

// ChatModel is the primary interface for interacting with language models.
// All LLM providers implement this interface, and the Router, middleware,
// and structured output layer all compose through it.
type ChatModel interface {
	// Generate sends a batch of messages and returns a complete AI response.
	Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error)

	// Stream sends a batch of messages and returns an iterator of response chunks.
	// Consumers should range over the returned sequence. A non-nil error terminates
	// the stream.
	Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error]

	// BindTools returns a new ChatModel that includes the given tool definitions
	// in every request. The original model is not modified.
	BindTools(tools []schema.ToolDefinition) ChatModel

	// ModelID returns the identifier of the underlying model (e.g. "gpt-4o").
	ModelID() string
}
