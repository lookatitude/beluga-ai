package messaging

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
)

// Global registry for messaging providers.
var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// Registry manages messaging provider registration and retrieval.
type Registry struct {
	creators map[string]func(context.Context, *Config) (iface.ConversationalBackend, error)
	mu       sync.RWMutex
}

// GetRegistry returns the global registry instance.
// It uses sync.Once to ensure thread-safe initialization.
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			creators: make(map[string]func(context.Context, *Config) (iface.ConversationalBackend, error)),
		}
	})
	return globalRegistry
}

// Register registers a new messaging provider with the registry.
func (r *Registry) Register(name string, creator func(context.Context, *Config) (iface.ConversationalBackend, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creators[name] = creator
}

// Create creates a new messaging backend instance using the registered provider.
func (r *Registry) Create(ctx context.Context, name string, config *Config) (iface.ConversationalBackend, error) {
	r.mu.RLock()
	creator, exists := r.creators[name]
	r.mu.RUnlock()

	if !exists {
		return nil, NewMessagingError("Create", ErrCodeProviderNotFound,
			fmt.Errorf("messaging provider '%s' not found", name))
	}

	// Validate configuration before creating backend
	if err := ValidateConfig(ctx, config); err != nil {
		return nil, NewMessagingError("Create", ErrCodeInvalidConfig, err)
	}

	// Set provider name in config if not already set
	if config.Provider == "" {
		config.Provider = name
	}

	return creator(ctx, config)
}

// ListProviders returns a list of all registered provider names.
func (r *Registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.creators))
	for name := range r.creators {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a provider is registered.
func (r *Registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.creators[name]
	return exists
}
