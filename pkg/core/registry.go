// Package core provides a registry pattern for DI container factories.
package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Static base errors for dynamic error wrapping (err113 compliance).
var (
	errContainerTypeNotRegistered = errors.New("container type not registered")
)

// ContainerCreatorFunc defines the function signature for creating containers.
type ContainerCreatorFunc func(ctx context.Context, opts ...DIOption) (Container, error)

// ContainerRegistry is the global registry for creating container instances.
// It maintains a registry of available container types and their creation functions.
type ContainerRegistry struct {
	creators map[string]ContainerCreatorFunc
	mu       sync.RWMutex
}

// NewContainerRegistry creates a new ContainerRegistry instance.
// The registry manages container type registration and creation following the factory pattern.
//
// Returns:
//   - *ContainerRegistry: A new container registry instance
//
// Example:
//
//	registry := core.NewContainerRegistry()
//	registry.Register("default", defaultContainerCreator)
func NewContainerRegistry() *ContainerRegistry {
	return &ContainerRegistry{
		creators: make(map[string]ContainerCreatorFunc),
	}
}

// Register registers a new container type with the registry.
// This method is thread-safe and allows extending the framework with custom container types.
//
// Parameters:
//   - containerType: Unique identifier for the container type (e.g., "default", "scoped")
//   - creator: Function that creates container instances of this type
//
// Example:
//
//	registry.Register("custom", func(ctx context.Context, opts ...core.DIOption) (core.Container, error) {
//	    return NewCustomContainer(opts...)
//	})
func (r *ContainerRegistry) Register(containerType string, creator ContainerCreatorFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creators[containerType] = creator
}

// Create creates a new container instance using the registered container type.
// This method is thread-safe and returns an error if the container type is not registered.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - containerType: Type of container to create (must be registered)
//   - opts: Optional configuration options for the container
//
// Returns:
//   - Container: A new container instance
//   - error: Error if container type is not registered or creation fails
//
// Example:
//
//	container, err := registry.Create(ctx, "default", core.WithLogger(logger))
//	if err != nil {
//	    log.Fatal(err)
//	}
func (r *ContainerRegistry) Create(ctx context.Context, containerType string, opts ...DIOption) (Container, error) {
	r.mu.RLock()
	creator, exists := r.creators[containerType]
	r.mu.RUnlock()

	if !exists {
		return nil, NewFrameworkError(
			"create_container",
			ErrorCodeNotFound,
			fmt.Sprintf("container type '%s' not registered", containerType),
			fmt.Errorf("%w: %s", errContainerTypeNotRegistered, containerType),
		)
	}
	return creator(ctx, opts...)
}

// ListContainerTypes returns a list of all registered container type names.
// This method is thread-safe and returns an empty slice if no types are registered.
//
// Returns:
//   - []string: Slice of registered container type names
//
// Example:
//
//	types := registry.ListContainerTypes()
//	fmt.Printf("Available container types: %v\n", types)
func (r *ContainerRegistry) ListContainerTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.creators))
	for name := range r.creators {
		names = append(names, name)
	}
	return names
}

// Global registry instance for easy access.
var globalContainerRegistry = NewContainerRegistry()

// RegisterContainerType registers a container type with the global registry.
// This is a convenience function for registering with the global registry.
//
// Parameters:
//   - containerType: Unique identifier for the container type
//   - creator: Function that creates container instances of this type
//
// Example:
//
//	core.RegisterContainerType("custom", customContainerCreator)
func RegisterContainerType(containerType string, creator ContainerCreatorFunc) {
	globalContainerRegistry.Register(containerType, creator)
}

// CreateContainer creates a container using the global registry.
// This is a convenience function for creating containers with the global registry.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - containerType: Type of container to create (must be registered)
//   - opts: Optional configuration options for the container
//
// Returns:
//   - Container: A new container instance
//   - error: Error if container type is not registered or creation fails
//
// Example:
//
//	container, err := core.CreateContainer(ctx, "default", core.WithLogger(logger))
func CreateContainer(ctx context.Context, containerType string, opts ...DIOption) (Container, error) {
	return globalContainerRegistry.Create(ctx, containerType, opts...)
}

// ListAvailableContainerTypes returns all available container types from the global registry.
// This is a convenience function for listing types from the global registry.
//
// Returns:
//   - []string: Slice of available container type names
//
// Example:
//
//	types := core.ListAvailableContainerTypes()
//	fmt.Printf("Available types: %v\n", types)
func ListAvailableContainerTypes() []string {
	return globalContainerRegistry.ListContainerTypes()
}

// GetRegistry returns the global registry instance.
// This follows the standard pattern used across all Beluga AI packages.
//
// Example:
//
//	registry := core.GetRegistry()
//	containerTypes := registry.ListContainerTypes()
func GetRegistry() *ContainerRegistry {
	return globalContainerRegistry
}

// Built-in container type constants.
const (
	ContainerTypeDefault = "default"
)

// init registers the built-in container types.
func init() {
	// Register built-in container types
	RegisterContainerType(ContainerTypeDefault, createDefaultContainer)
}

// Built-in container creators.
func createDefaultContainer(_ context.Context, opts ...DIOption) (Container, error) {
	return NewContainerWithOptions(opts...), nil
}
