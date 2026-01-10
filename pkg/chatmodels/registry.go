package chatmodels

import (
	"errors"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

// ChatModelFactory defines the function signature for creating chat models.
type ChatModelFactory func(model string, config *Config, options *iface.Options) (iface.ChatModel, error)

// Registry manages chat model provider registration and retrieval.
type Registry struct {
	providers map[string]ChatModelFactory
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
//	registry := chatmodels.GetRegistry()
//	if registry.IsRegistered("openai") {
//	    model, err := registry.CreateProvider("gpt-4", config, options)
//	}
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			providers: make(map[string]ChatModelFactory),
		}
	})
	return globalRegistry
}

// Register registers a provider factory function.
func (r *Registry) Register(name string, factory ChatModelFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = factory
}

// GetProvider returns a provider factory for the given name.
func (r *Registry) GetProvider(name string) (ChatModelFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.providers[name]
	if !exists {
		return nil, NewChatModelError("GetProvider", "", name, ErrCodeProviderNotSupported,
			fmt.Errorf("provider '%s' not registered", name))
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

// CreateProvider creates a chat model using the registered provider factory.
func (r *Registry) CreateProvider(model string, config *Config, options *iface.Options) (iface.ChatModel, error) {
	providerName := config.DefaultProvider
	if providerName == "" {
		return nil, NewChatModelError("CreateProvider", model, "", ErrCodeConfigInvalid,
			errors.New("provider name is required"))
	}

	factory, err := r.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return factory(model, config, options)
}
