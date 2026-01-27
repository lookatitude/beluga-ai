// Package base provides base implementations for tools.
package base

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/tools/iface"
)

// BaseTool provides a default implementation of the Tool interface.
// It can be embedded in tool implementations to simplify implementing the interface.
type BaseTool struct {
	inputSchema any
	name        string
	description string
}

// NewBaseTool creates a new BaseTool with the given name, description, and input schema.
func NewBaseTool(name, description string, inputSchema any) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		inputSchema: inputSchema,
	}
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
func (b *BaseTool) Definition() iface.ToolDefinition {
	return iface.ToolDefinition{
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
func (b *BaseTool) Execute(_ context.Context, _ any) (any, error) {
	return nil, fmt.Errorf("Execute not implemented for tool '%s'", b.name)
}

// Batch implements parallel execution of multiple inputs.
// By default, it executes each input concurrently using goroutines.
func (b *BaseTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	errs := make([]error, len(inputs))
	var firstErr error
	var errMu sync.Mutex

	var wg sync.WaitGroup
	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, inp any) {
			defer wg.Done()
			result, err := b.Execute(ctx, inp)
			results[idx] = result
			if err != nil {
				errMu.Lock()
				errs[idx] = err
				if firstErr == nil {
					firstErr = fmt.Errorf("error in batch processing item %d: %w", idx, err)
				}
				errMu.Unlock()
			}
		}(i, input)
	}
	wg.Wait()

	return results, firstErr
}

// Ensure BaseTool implements the Tool interface.
var _ iface.Tool = (*BaseTool)(nil)
