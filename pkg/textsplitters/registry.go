package textsplitters

import (
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
)

// Global registry for text splitters.
var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// SplitterFactory creates a TextSplitter from configuration.
type SplitterFactory func(config map[string]any) (iface.TextSplitter, error)

// Registry manages splitter provider registration and retrieval.
type Registry struct {
	factories map[string]SplitterFactory
	mu        sync.RWMutex
}

// GetRegistry returns the global registry instance.
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			factories: make(map[string]SplitterFactory),
		}
	})
	return globalRegistry
}

// Register adds a splitter factory to the registry.
// Panics if name is already registered (call during init only).
func (r *Registry) Register(name string, factory SplitterFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.factories[name]; exists {
		panic(fmt.Sprintf("splitter '%s' is already registered", name))
	}
	r.factories[name] = factory
}

// Create instantiates a splitter by name with the given configuration.
// Returns ErrCodeNotFound if the splitter is not registered.
func (r *Registry) Create(name string, config map[string]any) (iface.TextSplitter, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, NewSplitterError("Create", ErrCodeNotFound, fmt.Sprintf("splitter '%s' not registered", name), nil)
	}

	return factory(config)
}

// List returns all registered splitter names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a splitter name is registered.
func (r *Registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.factories[name]
	return exists
}
