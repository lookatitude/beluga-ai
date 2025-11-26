package turndetection

import (
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection/iface"
)

// Global registry for Turn Detection providers.
var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// Registry manages Turn Detection provider registration and retrieval.
type Registry struct {
	providers map[string]func(*Config) (iface.TurnDetector, error)
	mu        sync.RWMutex
}

// GetRegistry returns the global registry instance.
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			providers: make(map[string]func(*Config) (iface.TurnDetector, error)),
		}
	})
	return globalRegistry
}

// Register registers a provider factory function.
func (r *Registry) Register(name string, factory func(*Config) (iface.TurnDetector, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = factory
}

// GetProvider returns a provider instance for the given name.
func (r *Registry) GetProvider(name string, config *Config) (iface.TurnDetector, error) {
	r.mu.RLock()
	factory, exists := r.providers[name]
	r.mu.RUnlock()

	if !exists {
		return nil, NewTurnDetectionError("GetProvider", ErrCodeUnsupportedProvider,
			fmt.Errorf("provider '%s' not registered", name))
	}

	return factory(config)
}

// ListProviders returns a list of all registered provider names.
func (r *Registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a provider is registered.
func (r *Registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.providers[name]
	return exists
}
