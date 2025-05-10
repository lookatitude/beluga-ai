package tools

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/schema" // For potential future use with structured IO schemas
)

// Tool defines the interface for a tool that an agent can use.
// Each tool must have a unique name, a description of its capabilities,
// a schema for its expected input, and an execution method.	ype Tool interface {
	// GetName returns the unique name of the tool.
	GetName() string
	// GetDescription returns a description of what the tool does, its inputs, and outputs.
	// This is used by the LLM to decide when and how to use the tool.
	GetDescription() string
	// GetInputSchema returns a JSON schema string defining the expected input format for the tool.
	// This helps in validating input and can also be provided to the LLM.
	GetInputSchema() string // JSON Schema as a string
	// Execute runs the tool with the given input and returns the output string or an error.
	// The input is currently a string; tools are responsible for parsing it if necessary
	// based on their input schema. Future versions might support structured input directly.
	Execute(ctx context.Context, input string) (string, error)
}

// BaseTool provides a common structure that can be embedded in concrete tool implementations.
// It helps in fulfilling parts of the Tool interface if common patterns emerge.
type BaseTool struct {
	Name        string
	Description string
	InputSchema string // JSON schema as a string
}

// GetName returns the tool's name.
func (bt *BaseTool) GetName() string {
	return bt.Name
}

// GetDescription returns the tool's description.
func (bt *BaseTool) GetDescription() string {
	return bt.Description
}

// GetInputSchema returns the tool's input schema.
func (bt *BaseTool) GetInputSchema() string {
	return bt.InputSchema
}

// Registry defines the interface for a tool registry.
// A registry allows for discovering and retrieving tools by name.	ype Registry interface {
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
		return fmt.Errorf("tool with name 	%s	 already registered", tool.GetName())
	}
	r.tools[tool.GetName()] = tool
	return nil
}

// GetTool retrieves a tool from the registry by its name.
func (r *InMemoryToolRegistry) GetTool(name string) (Tool, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool with name 	%s	 not found", name)
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
		descriptions = append(descriptions, fmt.Sprintf("- %s: %s (Input Schema: %s)", tool.GetName(), tool.GetDescription(), tool.GetInputSchema()))
	}
	return strings.Join(descriptions, "\n")
}

// Ensure InMemoryToolRegistry implements the Registry interface.
var _ Registry = (*InMemoryToolRegistry)(nil)

