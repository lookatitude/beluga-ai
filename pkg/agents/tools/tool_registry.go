package tools

import (
	"fmt"
	"strings"
	// "context" // Unused
	// "regexp" // Unused
	// "strconv" // Unused
	// "github.com/lookatitude/beluga-ai/pkg/schema" // Unused for now, tool.go handles schema related types
)

// Registry defines the interface for a tool registry.
// A registry allows for discovering and retrieving tools by name.
type Registry interface {
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
	if tool.GetName() == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if _, exists := r.tools[tool.GetName()]; exists {
		return fmt.Errorf("tool with name %s already registered", tool.GetName())
	}
	r.tools[tool.GetName()] = tool
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

// ListTools returns a list of names of all registered tools.
func (r *InMemoryToolRegistry) ListTools() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetToolDescriptions returns a formatted string of all tool names and descriptions.
// This can be used to provide context to an LLM about available tools.
func (r *InMemoryToolRegistry) GetToolDescriptions() string {
	var descriptions []string
	for _, tool := range r.tools {
		schemaStr, err := tool.GetInputSchemaString() // Use the new method
		if err != nil {
			schemaStr = fmt.Sprintf("[error getting schema: %v]", err) // Include error in description
		}
		descriptions = append(descriptions, fmt.Sprintf("- %s: %s (Input Schema: %s)", tool.GetName(), tool.GetDescription(), schemaStr))
	}
	return strings.Join(descriptions, "\n")
}

// Ensure InMemoryToolRegistry implements the Registry interface.
var _ Registry = (*InMemoryToolRegistry)(nil)

