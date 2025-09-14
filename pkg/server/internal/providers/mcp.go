// Package providers provides concrete implementations of REST and MCP servers.
package providers

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/server"
	"github.com/lookatitude/beluga-ai/pkg/server/iface"
)

// MCPProvider provides a ready-to-use MCP server implementation
type MCPProvider struct {
	server server.MCPServer
}

// NewMCPProvider creates a new MCP provider with default configuration
func NewMCPProvider(opts ...server.Option) (*MCPProvider, error) {
	// Set default MCP configuration if not provided
	hasMCPConfig := false
	for _, opt := range opts {
		// Check if MCP config is already provided
		_ = opt
		hasMCPConfig = true // Simplified check
	}

	if !hasMCPConfig {
		defaultOpts := []server.Option{
			server.WithMCPConfig(server.DefaultMCPConfig()),
		}
		opts = append(defaultOpts, opts...)
	}

	srv, err := NewMCPServer(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	return &MCPProvider{
		server: srv,
	}, nil
}

// Start starts the MCP server
func (p *MCPProvider) Start(ctx context.Context) error {
	return p.server.Start(ctx)
}

// Stop stops the MCP server
func (p *MCPProvider) Stop(ctx context.Context) error {
	return p.server.Stop(ctx)
}

// RegisterTool registers a tool with the MCP server
func (p *MCPProvider) RegisterTool(tool iface.MCPTool) error {
	return p.server.RegisterTool(tool)
}

// RegisterResource registers a resource with the MCP server
func (p *MCPProvider) RegisterResource(resource iface.MCPResource) error {
	return p.server.RegisterResource(resource)
}

// GetServer returns the underlying MCP server for advanced usage
func (p *MCPProvider) GetServer() server.MCPServer {
	return p.server
}

// CalculatorTool is an example MCP tool that performs basic calculations
type CalculatorTool struct{}

// NewCalculatorTool creates a new calculator tool
func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

// Name returns the tool name
func (t *CalculatorTool) Name() string {
	return "calculator"
}

// Description returns the tool description
func (t *CalculatorTool) Description() string {
	return "Performs basic arithmetic calculations"
}

// InputSchema returns the JSON schema for tool input
func (t *CalculatorTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"add", "subtract", "multiply", "divide"},
				"description": "The arithmetic operation to perform",
			},
			"a": map[string]interface{}{
				"type":        "number",
				"description": "First operand",
			},
			"b": map[string]interface{}{
				"type":        "number",
				"description": "Second operand",
			},
		},
		"required": []string{"operation", "a", "b"},
	}
}

// Execute performs the calculation
func (t *CalculatorTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	operation, ok := input["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid operation type")
	}

	a, ok := input["a"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid operand 'a' type")
	}

	b, ok := input["b"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid operand 'b' type")
	}

	var result float64
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}

	return map[string]interface{}{
		"operation": operation,
		"a":         a,
		"b":         b,
		"result":    result,
	}, nil
}

// FileResource is an example MCP resource that provides file access
type FileResource struct {
	name        string
	description string
	filePath    string
	mimeType    string
}

// NewFileResource creates a new file resource
func NewFileResource(name, description, filePath, mimeType string) *FileResource {
	return &FileResource{
		name:        name,
		description: description,
		filePath:    filePath,
		mimeType:    mimeType,
	}
}

// URI returns the resource URI
func (r *FileResource) URI() string {
	return fmt.Sprintf("file://%s", r.filePath)
}

// Name returns the resource name
func (r *FileResource) Name() string {
	return r.name
}

// Description returns the resource description
func (r *FileResource) Description() string {
	return r.description
}

// MimeType returns the resource MIME type
func (r *FileResource) MimeType() string {
	return r.mimeType
}

// Read reads the resource content
func (r *FileResource) Read(ctx context.Context) ([]byte, error) {
	// This is a simplified implementation
	// In a real implementation, you would read from the actual file
	return []byte(fmt.Sprintf("Content of file: %s", r.filePath)), nil
}

// TextResource is an example MCP resource that provides text content
type TextResource struct {
	name        string
	description string
	content     string
	mimeType    string
}

// NewTextResource creates a new text resource
func NewTextResource(name, description, content, mimeType string) *TextResource {
	return &TextResource{
		name:        name,
		description: description,
		content:     content,
		mimeType:    mimeType,
	}
}

// URI returns the resource URI
func (r *TextResource) URI() string {
	return fmt.Sprintf("text://%s", r.name)
}

// Name returns the resource name
func (r *TextResource) Name() string {
	return r.name
}

// Description returns the resource description
func (r *TextResource) Description() string {
	return r.description
}

// MimeType returns the resource MIME type
func (r *TextResource) MimeType() string {
	return r.mimeType
}

// Read reads the resource content
func (r *TextResource) Read(ctx context.Context) ([]byte, error) {
	return []byte(r.content), nil
}
