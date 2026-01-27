// Package tools provides implementations for tools that can be used by agents.
// It follows the Beluga AI Framework design patterns with consistent interfaces
// and a global registry pattern for tool type management.
//
// The package provides:
//   - A standardized Tool interface (iface.Tool)
//   - A global registry for tool type registration and creation
//   - Built-in tool implementations (API, Shell, GoFunc, MCP, Calculator, Echo)
//   - OTEL metrics and tracing for tool execution
//   - Standardized error handling with Op/Err/Code pattern
//
// Example usage:
//
//	// Create a calculator tool using the registry
//	tool, err := tools.CreateTool(ctx, tools.ToolTypeCalculator, tools.ToolConfig{
//	    Name:        "calc",
//	    Description: "A simple calculator",
//	    Type:        tools.ToolTypeCalculator,
//	})
//
//	// Execute the tool
//	result, err := tool.Execute(ctx, map[string]any{"expression": "2 + 2"})
package tools

import (
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/tools/iface"
	"github.com/lookatitude/beluga-ai/pkg/tools/internal/base"
)

// Tool is an alias for iface.Tool for convenience.
type Tool = core.Tool

// ToolDefinition is an alias for iface.ToolDefinition for convenience.
type ToolDefinition = core.ToolDefinition

// ToolRegistryInterface is an alias for iface.ToolRegistry.
type ToolRegistryInterface = iface.ToolRegistry

// StreamingTool is an alias for iface.StreamingTool.
type StreamingTool = iface.StreamingTool

// AsyncTool is an alias for iface.AsyncTool.
type AsyncTool = iface.AsyncTool

// InMemoryToolRegistry is an alias for iface.InMemoryToolRegistry.
type InMemoryToolRegistry = iface.InMemoryToolRegistry

// NewInMemoryToolRegistry creates a new InMemoryToolRegistry.
var NewInMemoryToolRegistry = iface.NewInMemoryToolRegistry

// BaseTool is an alias for the internal base.BaseTool for convenience.
// Embed this in custom tool implementations for default behavior.
type BaseTool = base.BaseTool

// NewBaseTool creates a new BaseTool with the given name, description, and input schema.
var NewBaseTool = base.NewBaseTool

// Core option compatibility types and functions.

// CoreOption is a core.Option for backward compatibility.
type CoreOption = core.Option

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

// WithConcurrency sets the max concurrency for batch operations.
func WithConcurrency(n int) core.Option {
	return newOption(func(config *map[string]any) {
		(*config)["max_concurrency"] = n
	})
}

// WithTimeoutSeconds sets a timeout duration for tool execution.
func WithTimeoutSeconds(seconds float64) core.Option {
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
