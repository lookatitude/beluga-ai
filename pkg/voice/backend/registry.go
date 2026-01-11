package backend

import (
	"context"
	"fmt"
	"sync"

	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// Global registry for voice backend providers.
var (
	globalRegistry *BackendRegistry
	registryOnce   sync.Once
)

// BackendRegistry manages voice backend provider registration and retrieval.
type BackendRegistry struct {
	creators map[string]func(context.Context, *vbiface.Config) (vbiface.VoiceBackend, error)
	mu       sync.RWMutex
}

// GetRegistry returns the global registry instance.
// It uses sync.Once to ensure thread-safe initialization.
func GetRegistry() *BackendRegistry {
	registryOnce.Do(func() {
		globalRegistry = &BackendRegistry{
			creators: make(map[string]func(context.Context, *vbiface.Config) (vbiface.VoiceBackend, error)),
		}
	})
	return globalRegistry
}

// Register registers a new voice backend provider with the registry.
func (r *BackendRegistry) Register(name string, creator func(context.Context, *vbiface.Config) (vbiface.VoiceBackend, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creators[name] = creator
}

// Create creates a new voice backend instance using the registered provider.
func (r *BackendRegistry) Create(ctx context.Context, name string, config *vbiface.Config) (vbiface.VoiceBackend, error) {
	r.mu.RLock()
	creator, exists := r.creators[name]
	r.mu.RUnlock()

	if !exists {
		return nil, NewBackendError("Create", ErrCodeProviderNotFound,
			fmt.Errorf("voice backend provider '%s' not found", name))
	}

	// Validate configuration before creating backend
	if err := ValidateConfig(config); err != nil {
		return nil, NewBackendError("Create", ErrCodeInvalidConfig, err)
	}

	// Set provider name in config if not already set
	if config.Provider == "" {
		config.Provider = name
	}

	return creator(ctx, config)
}

// ListProviders returns a list of all registered provider names.
func (r *BackendRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.creators))
	for name := range r.creators {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a provider is registered.
func (r *BackendRegistry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.creators[name]
	return exists
}

// GetProvider returns the factory function for a registered provider.
func (r *BackendRegistry) GetProvider(name string) (func(context.Context, *vbiface.Config) (vbiface.VoiceBackend, error), error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	creator, exists := r.creators[name]
	if !exists {
		return nil, NewBackendError("GetProvider", ErrCodeProviderNotFound,
			fmt.Errorf("voice backend provider '%s' not found", name))
	}
	return creator, nil
}
