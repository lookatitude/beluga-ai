// Package iface defines the registry interface for chat model providers.
// This contains factory types and registry interfaces used by providers
// to register themselves without importing the main chatmodels package.
package iface

import (
	"errors"
	"fmt"
	"sync"
)

// ProviderFactory defines the function signature for creating chat models.
// This type is used by providers to register themselves with the registry.
// The config parameter is `any` to avoid import cycles - providers should
// assert it to *chatmodels.Config when implementing.
type ProviderFactory func(model string, config any, options *Options) (ChatModel, error)

// Registry defines the interface for chat model provider registration.
// Implementations of this interface manage provider registration and creation.
type Registry interface {
	// Register registers a provider factory function with the given name.
	Register(name string, factory ProviderFactory)

	// GetProvider returns a provider factory for the given name.
	GetProvider(name string) (ProviderFactory, error)

	// CreateProvider creates a chat model using the registered provider factory.
	CreateProvider(model string, config any, options *Options) (ChatModel, error)

	// IsRegistered checks if a provider is registered.
	IsRegistered(name string) bool

	// ListProviders returns a list of all registered provider names.
	ListProviders() []string
}

// registry implements the Registry interface.
// This is an internal implementation that manages provider registration.
type registry struct {
	providers map[string]ProviderFactory
	mu        sync.RWMutex
}

// Global registry instance for easy access.
var (
	globalRegistry *registry
	registryOnce   sync.Once
)

// GetRegistry returns the global registry instance.
// This follows the standard pattern used across all Beluga AI packages.
// It uses sync.Once to ensure thread-safe initialization.
//
// Example:
//
//	registry := iface.GetRegistry()
//	if registry.IsRegistered("openai") {
//	    model, err := registry.CreateProvider("gpt-4", config, options)
//	}
func GetRegistry() Registry {
	registryOnce.Do(func() {
		globalRegistry = &registry{
			providers: make(map[string]ProviderFactory),
		}
	})
	return globalRegistry
}

// Register registers a provider factory function.
func (r *registry) Register(name string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = factory
}

// GetProvider returns a provider factory for the given name.
func (r *registry) GetProvider(name string) (ProviderFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not registered", name)
	}

	return factory, nil
}

// ListProviders returns a list of all registered provider names.
func (r *registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a provider is registered.
func (r *registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.providers[name]
	return exists
}

// ConfigProvider is an interface for getting the default provider name.
// This avoids importing the main chatmodels package for registry operations.
type ConfigProvider interface {
	GetDefaultProvider() string
}

// CreateProvider creates a chat model using the registered provider factory.
// The config parameter should implement ConfigProvider to provide the provider name.
func (r *registry) CreateProvider(model string, config any, options *Options) (ChatModel, error) {
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
