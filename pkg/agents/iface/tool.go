package iface

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/core"
)

// Tool defines the interface for tools that can be used by agents.
type Tool = core.Tool

// ToolDefinition provides metadata about a tool for LLM consumption.
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
