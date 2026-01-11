package documentloaders

import (
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
)

// Global registry for document loaders.
var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// LoaderFactory creates a DocumentLoader from configuration.
type LoaderFactory func(config map[string]any) (iface.DocumentLoader, error)

// Registry manages loader provider registration and retrieval.
type Registry struct {
	factories map[string]LoaderFactory
	mu        sync.RWMutex
}

// GetRegistry returns the global registry instance.
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			factories: make(map[string]LoaderFactory),
		}
	})
	return globalRegistry
}

// Register adds a loader factory to the registry.
// Panics if name is already registered (call during init only).
func (r *Registry) Register(name string, factory LoaderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.factories[name]; exists {
		panic(fmt.Sprintf("loader '%s' is already registered", name))
	}
	r.factories[name] = factory
}

// Create instantiates a loader by name with the given configuration.
// Returns ErrCodeNotFound if the loader is not registered.
func (r *Registry) Create(name string, config map[string]any) (iface.DocumentLoader, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, NewLoaderError("Create", ErrCodeNotFound, "", fmt.Sprintf("loader '%s' not registered", name), nil)
	}

	return factory(config)
}

// List returns all registered loader names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a loader name is registered.
func (r *Registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.factories[name]
	return exists
}
