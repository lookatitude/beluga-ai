package registry

import (
	"errors"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	registryiface "github.com/lookatitude/beluga-ai/pkg/chatmodels/registry/iface"
)

// Registry manages chat model provider registration and retrieval.
// This implements the registryiface.Registry interface.
type Registry struct {
	providers map[string]registryiface.ChatModelFactory
	mu        sync.RWMutex
}

// Global registry instance for easy access.
var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// GetRegistry returns the global registry instance.
// This follows the standard pattern used across all Beluga AI packages.
// It uses sync.Once to ensure thread-safe initialization.
//
// Example:
//
//	registry := registry.GetRegistry()
//	if registry.IsRegistered("openai") {
//	    model, err := registry.CreateProvider("gpt-4", config, options)
//	}
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			providers: make(map[string]registryiface.ChatModelFactory),
		}
	})
	return globalRegistry
}

// Register registers a provider factory function.
func (r *Registry) Register(name string, factory registryiface.ChatModelFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = factory
}

// GetProvider returns a provider factory for the given name.
func (r *Registry) GetProvider(name string) (registryiface.ChatModelFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not registered", name)
	}

	return factory, nil
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

// ConfigProvider is an interface for getting the default provider name.
// This avoids importing the main chatmodels package.
type ConfigProvider interface {
	GetDefaultProvider() string
}

// CreateProvider creates a chat model using the registered provider factory.
// The config parameter should implement ConfigProvider to provide the provider name.
func (r *Registry) CreateProvider(model string, config any, options *iface.Options) (iface.ChatModel, error) {
	// Extract provider name using the ConfigProvider interface
	cfgProvider, ok := config.(ConfigProvider)
	if !ok {
		return nil, errors.New("config must implement GetDefaultProvider() method")
	}

	providerName := cfgProvider.GetDefaultProvider()
	if providerName == "" {
		return nil, errors.New("provider name is required")
	}

	factory, err := r.GetProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider '%s': %w", providerName, err)
	}

	return factory(model, config, options)
}
