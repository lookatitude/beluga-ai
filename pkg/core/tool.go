package core

import (
	"context"
)

// Tool defines the interface for tools that can be used by agents and LLMs.
type Tool interface {
	// Name returns the unique identifier for this tool.
	Name() string

	// Description returns a human-readable description of what this tool does.
	Description() string

	// Definition returns the complete tool definition including schema, name, and description.
	Definition() ToolDefinition

	// Execute runs the tool with the given input.
	Execute(ctx context.Context, input any) (any, error)

	// Batch executes multiple inputs in parallel when possible.
	Batch(ctx context.Context, inputs []any) ([]any, error)
}

// ToolDefinition provides metadata about a tool for LLM consumption.
type ToolDefinition struct {
	InputSchema any    `json:"input_schema"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
