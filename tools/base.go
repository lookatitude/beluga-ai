// Package tools defines the interface for tools that agents can use.
package tools

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/core"
)

// ToolDefinition describes a tool, including its name, description, and input/output schemas.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema,omitempty"` // JSON Schema for input
	// TODO: Add OutputSchema?
}

// Tool is the interface that tools must implement.
type Tool interface {
	// Definition returns the tool's definition.
	Definition() ToolDefinition

	// Execute runs the tool with the given input.
	// Input can be any type, but specific tools will expect certain types (e.g., string, map[string]any).
	// Output can be any type, but often tools return strings or structured data (maps, structs).
	Execute(ctx context.Context, input any) (any, error)

	// Batch executes the tool for multiple inputs.
	// Provides a default sequential implementation if not overridden.
	Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error)
}

// BaseTool provides a default implementation for the Batch method.
type BaseTool struct{}

// Batch provides a default sequential implementation.
func (t *BaseTool) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	// This requires the concrete type implementing BaseTool to also implement Execute.
	// We use a type assertion here, which is a bit fragile. A better design might involve
	// embedding or requiring the Execute method differently.
	executor, ok := any(t).(interface {
		Execute(ctx context.Context, input any) (any, error)
	})
	if !ok {
		// This should not happen if BaseTool is embedded correctly in a type implementing Tool.
		return nil, fmt.Errorf("tool does not implement Execute method correctly")
	}

	results := make([]any, len(inputs))
	var firstErr error
	for i, input := range inputs {
		// TODO: Consider concurrency options from core.Option
		output, err := executor.Execute(ctx, input)
		// Collect first error but continue processing other inputs
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("error processing input %d: %w", i, err)
		}
		results[i] = output
	}
	return results, firstErr
}

// ToolRegistry holds available tools.
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(tool Tool) error {
	name := tool.Definition().Name
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool with name 	%s	 already registered", name)
	}
	r.tools[name] = tool
	return nil
}

// Get retrieves a tool by name.
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tools.
func (r *ToolRegistry) List() []Tool {
	list := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		list = append(list, tool)
	}
	return list
}
