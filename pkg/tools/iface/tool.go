// Package iface provides interface definitions for the tools package.
// It follows the Interface Segregation Principle by providing small, focused interfaces.
package iface

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/core"
)

// Tool defines the interface for tools that can be used by agents and LLMs.
// Tools are executable functions that agents can invoke to perform actions
// or retrieve information from external systems.
type Tool interface {
	// Name returns the unique identifier for this tool.
	Name() string

	// Description returns a human-readable description of what this tool does.
	Description() string

	// Definition returns the complete tool definition including schema, name, and description.
	Definition() core.ToolDefinition

	// Execute runs the tool with the given input.
	Execute(ctx context.Context, input any) (any, error)

	// Batch executes multiple inputs in parallel when possible.
	Batch(ctx context.Context, inputs []any) ([]any, error)
}

// ToolDefinition is an alias for core.ToolDefinition for backward compatibility.
type ToolDefinition = core.ToolDefinition

// ToolRegistry defines the interface for a tool registry.
// A registry allows for discovering and retrieving tools by name.
type ToolRegistry interface {
	RegisterTool(tool Tool) error
	GetTool(name string) (Tool, error)
	ListTools() []string
	GetToolDescriptions() string // Helper to get a formatted string of all tool names and descriptions
}

// InMemoryToolRegistry is a simple in-memory implementation of the ToolRegistry.
type InMemoryToolRegistry struct {
	tools map[string]Tool
}

// NewInMemoryToolRegistry creates a new InMemoryToolRegistry.
func NewInMemoryToolRegistry() *InMemoryToolRegistry {
	return &InMemoryToolRegistry{
		tools: make(map[string]Tool),
	}
}

// RegisterTool adds a tool to the registry.
func (r *InMemoryToolRegistry) RegisterTool(tool Tool) error {
	if tool.Name() == "" {
		return errors.New("tool name cannot be empty")
	}
	if _, exists := r.tools[tool.Name()]; exists {
		return fmt.Errorf("tool with name %s already exists", tool.Name())
	}
	r.tools[tool.Name()] = tool
	return nil
}

// GetTool retrieves a tool from the registry by its name.
func (r *InMemoryToolRegistry) GetTool(name string) (Tool, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool with name %s not found", name)
	}
	return tool, nil
}

// ListTools returns a list of all tool names in the registry.
func (r *InMemoryToolRegistry) ListTools() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetToolDescriptions returns a formatted string of all tool names and descriptions.
func (r *InMemoryToolRegistry) GetToolDescriptions() string {
	if len(r.tools) == 0 {
		return "No tools registered."
	}

	var descriptions []string
	for _, tool := range r.tools {
		descriptions = append(descriptions, fmt.Sprintf("- %s: %s", tool.Name(), tool.Description()))
	}

	return "Available tools:\n" + strings.Join(descriptions, "\n")
}

// StreamingTool extends Tool with streaming capabilities.
type StreamingTool interface {
	Tool
	// Stream returns a channel that yields partial results as they become available.
	Stream(ctx context.Context, input any) (<-chan any, error)
}

// AsyncTool extends Tool with async execution capabilities.
type AsyncTool interface {
	Tool
	// ExecuteAsync starts execution and returns immediately with a handle.
	ExecuteAsync(ctx context.Context, input any) (ToolExecution, error)
}

// ToolExecution represents an async tool execution in progress.
type ToolExecution interface {
	// ID returns the execution ID.
	ID() string
	// Status returns the current execution status.
	Status() ExecutionStatus
	// Result blocks until execution completes and returns the result.
	Result(ctx context.Context) (any, error)
	// Cancel cancels the execution.
	Cancel() error
}

// ExecutionStatus represents the status of an async tool execution.
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCanceled  ExecutionStatus = "canceled"
)
