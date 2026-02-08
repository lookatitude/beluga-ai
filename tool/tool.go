// Package tool provides the tool system for the Beluga AI framework.
// It defines the Tool interface, a type-safe FuncTool wrapper using generics,
// a thread-safe tool registry, middleware composition, hooks for lifecycle
// management, and an MCP client for connecting to remote tool servers.
//
// Usage:
//
//	// Define a typed input struct
//	type CalcInput struct {
//	    Expression string `json:"expression" description:"Math expression" required:"true"`
//	}
//
//	// Create a FuncTool that auto-generates JSON Schema
//	calc := tool.NewFuncTool("calculate", "Evaluate math expressions",
//	    func(ctx context.Context, input CalcInput) (*tool.Result, error) {
//	        return tool.TextResult("42"), nil
//	    },
//	)
//
//	// Add to a registry
//	reg := tool.NewRegistry()
//	reg.Add(calc)
package tool

import (
	"context"

	"github.com/lookatitude/beluga-ai/schema"
)

// Tool is the interface that all tools implement. A tool has a name, description,
// JSON Schema for its input parameters, and an Execute method that performs the
// tool's action given a parsed input map.
type Tool interface {
	// Name returns the unique identifier for this tool.
	Name() string

	// Description returns a human-readable description of what the tool does.
	// This is provided to the LLM to help it decide when to use the tool.
	Description() string

	// InputSchema returns a JSON Schema (as a map) describing the expected
	// input parameters for this tool.
	InputSchema() map[string]any

	// Execute runs the tool with the given input parameters and returns a
	// multimodal result. The input map corresponds to the InputSchema.
	Execute(ctx context.Context, input map[string]any) (*Result, error)
}

// Result holds the output of a tool execution. Content is multimodal,
// supporting text, images, audio, and other content types via schema.ContentPart.
type Result struct {
	// Content holds the result content parts returned by the tool.
	Content []schema.ContentPart

	// IsError indicates whether the tool execution resulted in an error.
	// When true, the content typically contains an error description.
	IsError bool
}

// TextResult creates a Result containing a single text content part.
func TextResult(text string) *Result {
	return &Result{
		Content: []schema.ContentPart{
			schema.TextPart{Text: text},
		},
	}
}

// ErrorResult creates a Result from an error, marking IsError as true.
// The error message is placed in a text content part.
func ErrorResult(err error) *Result {
	return &Result{
		Content: []schema.ContentPart{
			schema.TextPart{Text: err.Error()},
		},
		IsError: true,
	}
}

// ToDefinition converts a Tool to a schema.ToolDefinition suitable for
// sending to an LLM provider.
func ToDefinition(t Tool) schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		InputSchema: t.InputSchema(),
	}
}
