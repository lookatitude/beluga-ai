// Package tools provides test utilities for tool testing.
package tools

import (
	"context"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/tools/iface"
)

// MockTool is a mock implementation of the Tool interface for testing.
type MockTool struct {
	ToolName        string
	ToolDescription string
	ToolSchema      any
	ExecuteFunc     func(ctx context.Context, input any) (any, error)
	BatchFunc       func(ctx context.Context, inputs []any) ([]any, error)
	ExecuteCalls    int
	BatchCalls      int
}

// NewMockTool creates a new MockTool with the given name and description.
func NewMockTool(name, description string) *MockTool {
	return &MockTool{
		ToolName:        name,
		ToolDescription: description,
		ToolSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type": "string",
				},
			},
		},
	}
}

// Name returns the tool's name.
func (m *MockTool) Name() string {
	return m.ToolName
}

// Description returns the tool's description.
func (m *MockTool) Description() string {
	return m.ToolDescription
}

// Definition returns the tool's definition.
func (m *MockTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        m.ToolName,
		Description: m.ToolDescription,
		InputSchema: m.ToolSchema,
	}
}

// Execute executes the mock tool.
func (m *MockTool) Execute(ctx context.Context, input any) (any, error) {
	m.ExecuteCalls++
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, input)
	}
	return "mock result", nil
}

// Batch executes multiple inputs.
func (m *MockTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	m.BatchCalls++
	if m.BatchFunc != nil {
		return m.BatchFunc(ctx, inputs)
	}
	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = "mock result"
	}
	return results, nil
}

// WithExecute sets the execute function for the mock.
func (m *MockTool) WithExecute(fn func(ctx context.Context, input any) (any, error)) *MockTool {
	m.ExecuteFunc = fn
	return m
}

// WithBatch sets the batch function for the mock.
func (m *MockTool) WithBatch(fn func(ctx context.Context, inputs []any) ([]any, error)) *MockTool {
	m.BatchFunc = fn
	return m
}

// WithSchema sets the input schema for the mock.
func (m *MockTool) WithSchema(schema any) *MockTool {
	m.ToolSchema = schema
	return m
}

// Ensure MockTool implements the Tool interface.
var _ iface.Tool = (*MockTool)(nil)

// MockToolRegistry is a mock implementation of ToolRegistry for testing.
type MockToolRegistry struct {
	Tools map[string]iface.Tool
}

// NewMockToolRegistry creates a new MockToolRegistry.
func NewMockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		Tools: make(map[string]iface.Tool),
	}
}

// RegisterTool registers a tool.
func (r *MockToolRegistry) RegisterTool(tool iface.Tool) error {
	r.Tools[tool.Name()] = tool
	return nil
}

// GetTool retrieves a tool by name.
func (r *MockToolRegistry) GetTool(name string) (iface.Tool, error) {
	tool, ok := r.Tools[name]
	if !ok {
		return nil, NewNotFoundError("get_tool", name)
	}
	return tool, nil
}

// ListTools returns all tool names.
func (r *MockToolRegistry) ListTools() []string {
	names := make([]string, 0, len(r.Tools))
	for name := range r.Tools {
		names = append(names, name)
	}
	return names
}

// GetToolDescriptions returns formatted tool descriptions.
func (r *MockToolRegistry) GetToolDescriptions() string {
	if len(r.Tools) == 0 {
		return "No tools registered."
	}
	parts := []string{"Available tools:"}
	for _, tool := range r.Tools {
		parts = append(parts, "- "+tool.Name()+": "+tool.Description())
	}
	return strings.Join(parts, "\n") + "\n"
}

// Ensure MockToolRegistry implements ToolRegistry.
var _ iface.ToolRegistry = (*MockToolRegistry)(nil)
