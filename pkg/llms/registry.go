package llms

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// Global registry for LLM providers.
var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// Registry manages LLM provider registration and retrieval.
type Registry struct {
	providerFactories map[string]func(*Config) (iface.ChatModel, error)
	llmFactories      map[string]func(*Config) (iface.LLM, error)
	mu                sync.RWMutex
}

// GetRegistry returns the global registry instance.
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			providerFactories: make(map[string]func(*Config) (iface.ChatModel, error)),
			llmFactories:      make(map[string]func(*Config) (iface.LLM, error)),
		}
	})
	return globalRegistry
}

// Register registers a provider factory function.
func (r *Registry) Register(name string, factory func(*Config) (iface.ChatModel, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providerFactories[name] = factory
}

// RegisterLLM registers an LLM factory function.
func (r *Registry) RegisterLLM(name string, factory func(*Config) (iface.LLM, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llmFactories[name] = factory
}

// GetProvider returns a provider instance for the given name.
func (r *Registry) GetProvider(name string, config *Config) (iface.ChatModel, error) {
	r.mu.RLock()
	factory, exists := r.providerFactories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, NewLLMError("GetProvider", ErrCodeUnsupportedProvider,
			fmt.Errorf("provider '%s' not registered", name))
	}

	// Set provider name in config if not already set
	if config.Provider == "" {
		config.Provider = name
	}

	return factory(config)
}

// GetLLM returns an LLM instance for the given name.
func (r *Registry) GetLLM(name string, config *Config) (iface.LLM, error) {
	r.mu.RLock()
	factory, exists := r.llmFactories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, NewLLMError("GetLLM", ErrCodeUnsupportedProvider,
			fmt.Errorf("LLM '%s' not registered", name))
	}

	// Set provider name in config if not already set
	if config.Provider == "" {
		config.Provider = name
	}

	return factory(config)
}

// ListProviders returns a list of all registered provider names.
func (r *Registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providerFactories))
	for name := range r.providerFactories {
		names = append(names, name)
	}
	return names
}

// ListLLMs returns a list of all registered LLM names.
func (r *Registry) ListLLMs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.llmFactories))
	for name := range r.llmFactories {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a provider is registered.
func (r *Registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.providerFactories[name]
	return exists
}

// IsLLMRegistered checks if an LLM is registered.
func (r *Registry) IsLLMRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.llmFactories[name]
	return exists
}

// NewProvider creates a new LLM provider instance based on the configuration.
// It uses the global registry to find and instantiate the appropriate provider.
func NewProvider(ctx context.Context, providerName string, config *Config, opts ...ConfigOption) (iface.ChatModel, error) {
	// Apply options
	if config == nil {
		config = DefaultConfig()
	}
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewLLMError("NewProvider", ErrCodeInvalidConfig, err)
	}

	// Override provider name from parameter if provided
	if providerName != "" {
		config.Provider = providerName
	}

	// Get provider from registry
	registry := GetRegistry()
	provider, err := registry.GetProvider(config.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider '%s': %w", config.Provider, err)
	}

	return provider, nil
}
