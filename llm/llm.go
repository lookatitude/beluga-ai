// Package llm provides the LLM abstraction layer for the Beluga AI framework.
// It defines the ChatModel interface that all LLM providers implement, a
// provider registry for dynamic instantiation, composable middleware,
// lifecycle hooks, structured output parsing, context window management,
// tokenization, rate limiting, and an LLM router for multi-backend routing.
//
// Providers register themselves via init() so that importing a provider
// package is sufficient to make it available through the registry:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
//
//	model, err := llm.New("openai", cfg)
//
// Middleware wraps ChatModel to add cross-cutting concerns:
//
//	model = llm.ApplyMiddleware(model, llm.WithLogging(logger), llm.WithFallback(backup))
//
// Streaming uses iter.Seq2 (Go 1.23+):
//
//	for chunk, err := range model.Stream(ctx, msgs) {
//	    if err != nil { break }
//	    fmt.Print(chunk.Delta)
//	}
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
