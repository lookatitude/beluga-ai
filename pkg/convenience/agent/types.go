package agent

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/core"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
)

// Agent defines the simplified interface for convenience agents.
// It provides a streamlined API for running agents with optional memory and tools.
type Agent interface {
	// Run executes the agent with a simple string input and returns a string response.
	// This is the most common use case for agent interaction.
	// If memory is configured, it automatically loads and saves conversation context.
	Run(ctx context.Context, input string) (string, error)

	// RunWithInputs executes the agent with a map of inputs.
	// This provides more flexibility for complex agent interactions.
	// The output map typically contains an "output" key with the response.
	RunWithInputs(ctx context.Context, inputs map[string]any) (map[string]any, error)

	// Stream executes the agent and returns a channel that streams response chunks.
	// The channel is closed when the response is complete or an error occurs.
	// Errors are sent on the channel and should be checked.
	Stream(ctx context.Context, input string) (<-chan StreamChunk, error)

	// GetName returns the name of the agent.
	GetName() string

	// GetTools returns the tools available to the agent.
	GetTools() []core.Tool

	// GetMemory returns the memory instance if configured, nil otherwise.
	GetMemory() memoryiface.Memory

	// Shutdown gracefully stops the agent and releases resources.
	Shutdown() error
}

// StreamChunk represents a chunk of streamed output.
type StreamChunk struct {
	// Content is the text content of this chunk.
	Content string

	// Error is set if an error occurred during streaming.
	Error error

	// Done indicates this is the final chunk.
	Done bool
}
