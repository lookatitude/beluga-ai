// Package tools provides a standardized registry pattern for tool creation.
// This follows the Beluga AI Framework design patterns with consistent factory interfaces.
package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/tools/iface"
)

// Static base errors for dynamic error wrapping (err113 compliance).
var (
	errToolTypeNotRegistered = errors.New("tool type not registered")
)

// ToolCreatorFunc defines the function signature for creating tools.
type ToolCreatorFunc func(ctx context.Context, config ToolConfig) (iface.Tool, error)

// ToolRegistry is the global registry for creating tool instances.
// It maintains a registry of available tool types and their creation functions.
type ToolRegistry struct {
	creators map[string]ToolCreatorFunc
	tools    map[string]iface.Tool
	mu       sync.RWMutex
}

// NewToolRegistry creates a new ToolRegistry instance.
// The registry manages tool type registration and creation following the factory pattern.
//
// Returns:
//   - *ToolRegistry: A new tool registry instance
//
// Example:
//
//	registry := tools.NewToolRegistry()
//	registry.RegisterType("calculator", calculatorCreator)
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		creators: make(map[string]ToolCreatorFunc),
		tools:    make(map[string]iface.Tool),
	}
}

// RegisterType registers a new tool type with the registry.
// This method is thread-safe and allows extending the framework with custom tool types.
//
// Parameters:
//   - toolType: Unique identifier for the tool type (e.g., "api", "shell", "calculator")
//   - creator: Function that creates tool instances of this type
//
// Example:
//
//	registry.RegisterType("custom", func(ctx context.Context, config tools.ToolConfig) (iface.Tool, error) {
//	    return NewCustomTool(config)
//	})
func (r *ToolRegistry) RegisterType(toolType string, creator ToolCreatorFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creators[toolType] = creator
}

// Create creates a new tool instance using the registered tool type.
// This method is thread-safe and returns an error if the tool type is not registered.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - toolType: Type of tool to create (must be registered)
//   - config: Tool configuration
//
// Returns:
//   - iface.Tool: A new tool instance
//   - error: Error if tool type is not registered or creation fails
//
// Example:
//
//	tool, err := registry.Create(ctx, "calculator", config)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (r *ToolRegistry) Create(ctx context.Context, toolType string, config ToolConfig) (iface.Tool, error) {
	r.mu.RLock()
	creator, exists := r.creators[toolType]
	r.mu.RUnlock()

	if !exists {
		return nil, NewToolError(
			"create_tool",
			ErrorCodeNotFound,
			fmt.Sprintf("tool type '%s' not registered", toolType),
			fmt.Errorf("%w: %s", errToolTypeNotRegistered, toolType),
		)
	}
	return creator(ctx, config)
}

// RegisterTool registers a tool instance with the registry.
// This allows pre-created tool instances to be stored and retrieved by name.
//
// Parameters:
//   - tool: The tool instance to register
//
// Returns:
//   - error: Error if tool name is empty or already registered
func (r *ToolRegistry) RegisterTool(tool iface.Tool) error {
	if tool.Name() == "" {
		return NewToolError("register_tool", ErrorCodeInvalidInput, "tool name cannot be empty", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name()]; exists {
		return NewAlreadyExistsError("register_tool", tool.Name())
	}

	r.tools[tool.Name()] = tool
	return nil
}

// GetTool retrieves a registered tool instance by name.
//
// Parameters:
//   - name: Name of the tool to retrieve
//
// Returns:
//   - iface.Tool: The registered tool instance
//   - error: Error if tool is not found
func (r *ToolRegistry) GetTool(name string) (iface.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, NewNotFoundError("get_tool", name)
	}
	return tool, nil
}

// ListToolTypes returns a list of all registered tool type names.
// This method is thread-safe and returns an empty slice if no types are registered.
//
// Returns:
//   - []string: Slice of registered tool type names
//
// Example:
//
//	types := registry.ListToolTypes()
//	fmt.Printf("Available tool types: %v\n", types)
func (r *ToolRegistry) ListToolTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.creators))
	for name := range r.creators {
		names = append(names, name)
	}
	return names
}

// ListTools returns a list of all registered tool names.
func (r *ToolRegistry) ListTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetToolDescriptions returns a formatted string of all tool names and descriptions.
func (r *ToolRegistry) GetToolDescriptions() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.tools) == 0 {
		return "No tools registered."
	}

	var parts []string
	parts = append(parts, "Available tools:")
	for _, tool := range r.tools {
		parts = append(parts, fmt.Sprintf("- %s: %s", tool.Name(), tool.Description()))
	}
	return strings.Join(parts, "\n") + "\n"
}

// Clear removes all registered tools and types.
func (r *ToolRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.creators = make(map[string]ToolCreatorFunc)
	r.tools = make(map[string]iface.Tool)
}

// Global registry instance for easy access.
var globalToolRegistry = NewToolRegistry()

// RegisterToolType registers a tool type with the global registry.
// This is a convenience function for registering with the global registry.
//
// Parameters:
//   - toolType: Unique identifier for the tool type
//   - creator: Function that creates tool instances of this type
//
// Example:
//
//	tools.RegisterToolType("custom", customToolCreator)
func RegisterToolType(toolType string, creator ToolCreatorFunc) {
	globalToolRegistry.RegisterType(toolType, creator)
}

// CreateTool creates a tool using the global registry.
// This is a convenience function for creating tools with the global registry.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - toolType: Type of tool to create (must be registered)
//   - config: Tool configuration
//
// Returns:
//   - iface.Tool: A new tool instance
//   - error: Error if tool type is not registered or creation fails
//
// Example:
//
//	tool, err := tools.CreateTool(ctx, "calculator", config)
func CreateTool(ctx context.Context, toolType string, config ToolConfig) (iface.Tool, error) {
	return globalToolRegistry.Create(ctx, toolType, config)
}

// RegisterTool registers a tool instance with the global registry.
func RegisterTool(tool iface.Tool) error {
	return globalToolRegistry.RegisterTool(tool)
}

// GetTool retrieves a tool from the global registry by name.
func GetTool(name string) (iface.Tool, error) {
	return globalToolRegistry.GetTool(name)
}

// ListAvailableToolTypes returns all available tool types from the global registry.
// This is a convenience function for listing types from the global registry.
//
// Returns:
//   - []string: Slice of available tool type names
//
// Example:
//
//	types := tools.ListAvailableToolTypes()
//	fmt.Printf("Available types: %v\n", types)
func ListAvailableToolTypes() []string {
	return globalToolRegistry.ListToolTypes()
}

// GetRegistry returns the global registry instance.
// This follows the standard pattern used across all Beluga AI packages.
//
// Example:
//
//	registry := tools.GetRegistry()
//	toolTypes := registry.ListToolTypes()
func GetRegistry() *ToolRegistry {
	return globalToolRegistry
}

// Built-in tool type constants.
const (
	ToolTypeAPI        = "api"
	ToolTypeShell      = "shell"
	ToolTypeGoFunc     = "gofunc"
	ToolTypeMCP        = "mcp"
	ToolTypeCalculator = "calculator"
	ToolTypeEcho       = "echo"
)

// Ensure ToolRegistry implements iface.ToolRegistry.
var _ iface.ToolRegistry = (*ToolRegistry)(nil)
