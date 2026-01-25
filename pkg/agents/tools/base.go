// Package tools defines implementations for tools that can be used by agents.
package tools

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/core"
)

// Tool is an alias for iface.Tool for backward compatibility.
type Tool = core.Tool

// ToolDefinition is an alias for iface.ToolDefinition for backward compatibility.
type ToolDefinition = core.ToolDefinition

// BaseTool provides a default implementation of the Tool interface.
// It can be embedded in tool implementations to simplify implementing the interface.
type BaseTool struct {
	inputSchema any
	name        string
	description string
}

// Name returns the tool's name.
func (b *BaseTool) Name() string {
	return b.name
}

// Description returns the tool's description.
func (b *BaseTool) Description() string {
	return b.description
}

// Definition returns the tool's definition.
func (b *BaseTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        b.name,
		Description: b.description,
		InputSchema: b.inputSchema,
	}
}

// SetName sets the tool's name.
func (b *BaseTool) SetName(name string) {
	b.name = name
}

// SetDescription sets the tool's description.
func (b *BaseTool) SetDescription(description string) {
	b.description = description
}

// SetInputSchema sets the tool's input schema.
func (b *BaseTool) SetInputSchema(schema any) {
	b.inputSchema = schema
}

// Execute is a placeholder implementation that must be overridden by concrete tool implementations.
func (b *BaseTool) Execute(ctx context.Context, input any) (any, error) {
	return nil, errors.New("Execute not implemented in base tool")
}

// Batch implements parallel execution of multiple inputs.
// By default, it executes each input sequentially. Override for specialized parallel implementations.
func (b *BaseTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	// Default implementation: process inputs sequentially
	results := make([]any, len(inputs))
	errs := make([]error, len(inputs))
	var firstErr error

	// Process inputs with basic concurrency
	var wg sync.WaitGroup
	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, inp any) {
			defer wg.Done()
			result, err := b.Execute(ctx, inp)
			results[idx] = result
			errs[idx] = err
		}(i, input)
	}
	wg.Wait()

	// Check for errors
	for i, err := range errs {
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("error in batch processing item %d: %w", i, err)
			}
		}
	}

	return results, firstErr
}

// funcOption implements core.Option.
type funcOption struct {
	f func(*map[string]any)
}

func (fo *funcOption) Apply(config *map[string]any) {
	fo.f(config)
}

// Helper function to create an option.
func newOption(f func(*map[string]any)) core.Option {
	return &funcOption{f: f}
}

// WithConcurrency sets the max concurrency for StructuredTool's Batch method.
func WithConcurrency(n int) core.Option {
	return newOption(func(config *map[string]any) {
		(*config)["max_concurrency"] = n
	})
}

// WithTimeout sets a timeout duration for tool execution.
func WithTimeout(seconds float64) core.Option {
	return newOption(func(config *map[string]any) {
		(*config)["timeout"] = seconds
	})
}

// WithRetries sets the number of times to retry a tool execution on failure.
func WithRetries(n int) core.Option {
	return newOption(func(config *map[string]any) {
		(*config)["retries"] = n
	})
}

// WithEmbedder provides an embedder to use with tools that require one.
func WithEmbedder(embedder any) core.Option {
	return newOption(func(config *map[string]any) {
		(*config)["embedder"] = embedder
	})
}

// WithK sets the number of items to retrieve.
func WithK(k int) core.Option {
	return newOption(func(config *map[string]any) {
		(*config)["k"] = k
	})
}

// WithFilter provides a metadata filter.
func WithFilter(filter map[string]any) core.Option {
	return newOption(func(config *map[string]any) {
		(*config)["filter"] = filter
	})
}
