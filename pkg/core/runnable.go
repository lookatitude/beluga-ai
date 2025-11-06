// Package core defines fundamental interfaces and types used throughout the Beluga-ai framework.
package core

import "context"

// Option defines the interface for configuration options passed to Runnable methods.
// Specific options (like Temperature, MaxTokens, Callbacks) will implement this interface
// to modify the behavior of Runnable components during execution.
type Option interface {
	// Apply modifies a configuration map. Implementations should add or update
	// key-value pairs relevant to their specific option.
	Apply(config *map[string]any)
}

// optionFunc is a helper type that allows an ordinary function to be used as an Option.
// This is a common pattern in Go for creating simple interface implementations.
type optionFunc func(config *map[string]any)

// Apply calls f(config), allowing optionFunc to satisfy the Option interface.
func (f optionFunc) Apply(config *map[string]any) {
	f(config)
}

// WithOption creates a new Option that sets a specific key-value pair in the configuration map.
// This is a convenient way to create ad-hoc options for Runnables.
func WithOption(key string, value any) Option {
	return optionFunc(func(config *map[string]any) {
		(*config)[key] = value
	})
}

// Runnable is the central abstraction in Beluga-ai, inspired by LangChain.
// It represents a component that can be invoked with input to produce output.
// Most components, including LLMs, PromptTemplates, Tools, Retrievers, Agents,
// Chains, and Graphs, are designed to implement this interface.
// This provides a unified way to compose and execute different parts of an AI application.
type Runnable interface {
	// Invoke executes the runnable component with a single input and returns a single output.
	// It is the primary method for synchronous execution.
	// `ctx` allows for cancellation and passing request-scoped values.
	// `input` is the data passed to the component.
	// `options` provide configuration specific to this invocation (e.g., LLM temperature).
	Invoke(ctx context.Context, input any, options ...Option) (any, error)

	// Batch executes the runnable component with multiple inputs concurrently or sequentially,
	// returning a corresponding list of outputs.
	// This can offer performance benefits for components that support batch processing.
	// `ctx` allows for cancellation.
	// `inputs` is a slice of input data.
	// `options` provide configuration for the batch execution.
	Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error)

	// Stream executes the runnable component with a single input and returns a channel
	// from which output chunks can be read asynchronously.
	// This is useful for components like LLMs that can produce output incrementally.
	// The channel should be closed by the implementation when the stream is complete.
	// Any error encountered during streaming should be sent as the last item on the channel
	// before closing it.
	// `ctx` allows for cancellation.
	// `input` is the data passed to the component.
	// `options` provide configuration for the streaming execution.
	Stream(ctx context.Context, input any, options ...Option) (<-chan any, error)

	// TODO: Consider adding Async versions or event streaming (like LangChain_s astream_events)
	//       for more fine-grained observability and control over asynchronous execution.
}
