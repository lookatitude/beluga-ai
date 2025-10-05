package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
)

// ConfigProviderRegistry implements a global registry for configuration provider management.
// It provides thread-safe provider registration, discovery, and creation capabilities
// following the Beluga AI Framework constitutional requirements.
type ConfigProviderRegistry struct {
	mu        sync.RWMutex
	creators  map[string]ProviderCreator
	providers map[string]iface.Provider
	metadata  map[string]ProviderMetadata
}

// ProviderCreator defines the function signature for creating configuration providers.
// It takes options and returns a Provider implementation or error.
type ProviderCreator func(options ProviderOptions) (iface.Provider, error)

// ProviderMetadata contains information about a configuration provider's capabilities.
type ProviderMetadata struct {
	// Basic Information
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Version     string `json:"version"`

	// Format Support
	SupportedFormats []string `json:"supported_formats" validate:"required"`
	DefaultFormat    string   `json:"default_format"`

	// Capabilities
	Capabilities    []string `json:"capabilities" validate:"required"`
	SupportsWatch   bool     `json:"supports_watch"`
	SupportsReload  bool     `json:"supports_reload"`
	SupportsEnvVars bool     `json:"supports_env_vars"`

	// Configuration Requirements
	RequiredOptions []string `json:"required_options"`
	OptionalOptions []string `json:"optional_options"`

	// Operational Metadata
	HealthCheckSupported bool          `json:"health_check_supported"`
	DefaultTimeout       time.Duration `json:"default_timeout"`
	MaxRetries           int           `json:"max_retries"`
	CacheSupported       bool          `json:"cache_supported"`
	DefaultCacheTTL      time.Duration `json:"default_cache_ttl"`

	// Provider-specific Information
	ProviderSpecific map[string]interface{} `json:"provider_specific,omitempty"`
}

// ProviderOptions represents the configuration options for creating a provider.
type ProviderOptions struct {
	// Provider Selection
	ProviderType string `json:"provider_type" validate:"required"`

	// Configuration Source
	ConfigName  string   `json:"config_name" validate:"required"`
	ConfigPaths []string `json:"config_paths" validate:"required,min=1"`
	EnvPrefix   string   `json:"env_prefix,omitempty"`

	// Format and Parsing
	Format           string `json:"format,omitempty"` // yaml, json, toml, auto
	AutoDetectFormat bool   `json:"auto_detect_format"`

	// Behavior Configuration
	EnableValidation bool          `json:"enable_validation"`
	EnableWatching   bool          `json:"enable_watching"`
	EnableCaching    bool          `json:"enable_caching"`
	CacheTTL         time.Duration `json:"cache_ttl,omitempty"`
	LoadTimeout      time.Duration `json:"load_timeout,omitempty"`
	MaxRetries       int           `json:"max_retries,omitempty"`

	// Provider-Specific Options
	ProviderSpecific map[string]interface{} `json:"provider_specific,omitempty"`

	// Observability
	EnableMetrics bool `json:"enable_metrics"`
	EnableTracing bool `json:"enable_tracing"`
	EnableLogging bool `json:"enable_logging"`
}

// Global registry instance - initialized lazily
var globalRegistry *ConfigProviderRegistry
var registryOnce sync.Once

// GetGlobalRegistry returns the global configuration provider registry instance.
// It initializes the registry on first access with thread-safe lazy initialization.
func GetGlobalRegistry() *ConfigProviderRegistry {
	registryOnce.Do(func() {
		globalRegistry = &ConfigProviderRegistry{
			creators:  make(map[string]ProviderCreator),
			providers: make(map[string]iface.Provider),
			metadata:  make(map[string]ProviderMetadata),
		}
	})
	return globalRegistry
}

// RegisterGlobal registers a configuration provider creator function globally.
// The creator function will be used to instantiate providers on demand.
// This is the primary method for registering providers in the global registry.
func RegisterGlobal(name string, creator ProviderCreator) error {
	if name == "" {
		return NewError("RegisterGlobal", ErrCodeInvalidProviderName, "", fmt.Errorf("provider name cannot be empty"))
	}
	if creator == nil {
		return NewError("RegisterGlobal", ErrCodeProviderCreationFailed, "", fmt.Errorf("provider creator cannot be nil"))
	}

	registry := GetGlobalRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, exists := registry.creators[name]; exists {
		return NewError("RegisterGlobal", ErrCodeProviderAlreadyRegistered, name, fmt.Errorf("provider %s already registered", name))
	}

	registry.creators[name] = creator
	return nil
}

// RegisterGlobalWithMetadata registers a provider with metadata for enhanced discovery.
func RegisterGlobalWithMetadata(name string, creator ProviderCreator, metadata ProviderMetadata) error {
	if err := RegisterGlobal(name, creator); err != nil {
		return err
	}

	registry := GetGlobalRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	metadata.Name = name
	registry.metadata[name] = metadata
	return nil
}

// NewRegistryProvider creates a new configuration provider instance from a registered creator.
// Returns error if provider is not registered or creation fails.
func NewRegistryProvider(ctx context.Context, name string, options ProviderOptions) (iface.Provider, error) {
	if name == "" {
		return nil, NewError("NewRegistryProvider", ErrCodeInvalidProviderName, "", fmt.Errorf("provider name cannot be empty"))
	}

	registry := GetGlobalRegistry()
	registry.mu.RLock()
	creator, exists := registry.creators[name]
	registry.mu.RUnlock()

	if !exists {
		return nil, NewError("NewRegistryProvider", ErrCodeProviderNotFound, name, fmt.Errorf("provider %s not registered", name))
	}

	// Validate provider options against metadata if available
	if err := registry.validateProviderOptions(name, options); err != nil {
		return nil, err
	}

	// Create provider with timeout context
	_, cancel := context.WithTimeout(ctx, registry.getCreationTimeout(name))
	defer cancel()

	provider, err := creator(options)
	if err != nil {
		return nil, NewError("NewRegistryProvider", ErrCodeProviderCreationFailed, name, fmt.Errorf("failed to create provider %s: %w", name, err))
	}

	// Cache provider for future use if caching is enabled
	if options.EnableCaching {
		registry.mu.Lock()
		registry.providers[name] = provider
		registry.mu.Unlock()
	}

	return provider, nil
}

// ListProviders returns a list of all registered provider names.
func ListProviders() []string {
	registry := GetGlobalRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	providers := make([]string, 0, len(registry.creators))
	for name := range registry.creators {
		providers = append(providers, name)
	}
	return providers
}

// GetProviderMetadata returns metadata for a registered provider.
func GetProviderMetadata(name string) (*ProviderMetadata, error) {
	registry := GetGlobalRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	metadata, exists := registry.metadata[name]
	if !exists {
		return nil, NewError("GetProviderMetadata", ErrCodeProviderNotFound, name, fmt.Errorf("provider metadata not found: %s", name))
	}

	// Return a copy to prevent external modification
	metadataCopy := metadata
	return &metadataCopy, nil
}

// IsProviderRegistered checks if a provider with the given name is registered.
func IsProviderRegistered(name string) bool {
	registry := GetGlobalRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	_, exists := registry.creators[name]
	return exists
}

// UnregisterProvider removes a provider from the registry.
// Returns error if provider is not registered or cannot be removed safely.
func UnregisterProvider(name string) error {
	registry := GetGlobalRegistry()
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, exists := registry.creators[name]; !exists {
		return NewError("UnregisterProvider", ErrCodeProviderNotFound, name, fmt.Errorf("provider %s not registered", name))
	}

	// Remove from all maps
	delete(registry.creators, name)
	delete(registry.providers, name)
	delete(registry.metadata, name)

	return nil
}

// GetProvidersForFormat returns providers that support a specific configuration format.
func GetProvidersForFormat(format string) ([]string, error) {
	if format == "" {
		return nil, NewError("GetProvidersForFormat", ErrCodeFormatNotSupported, "", fmt.Errorf("format cannot be empty"))
	}

	registry := GetGlobalRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	var providers []string
	for name, metadata := range registry.metadata {
		for _, supportedFormat := range metadata.SupportedFormats {
			if supportedFormat == format {
				providers = append(providers, name)
				break
			}
		}
	}

	return providers, nil
}

// GetProviderByCapability returns providers that support a specific capability.
func GetProviderByCapability(capability string) ([]string, error) {
	if capability == "" {
		return nil, NewError("GetProviderByCapability", ErrCodeInvalidProviderName, "", fmt.Errorf("capability cannot be empty"))
	}

	registry := GetGlobalRegistry()
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	var providers []string
	for name, metadata := range registry.metadata {
		for _, cap := range metadata.Capabilities {
			if cap == capability {
				providers = append(providers, name)
				break
			}
		}
	}

	return providers, nil
}

// validateProviderOptions validates provider options against metadata requirements.
func (r *ConfigProviderRegistry) validateProviderOptions(providerName string, options ProviderOptions) error {
	metadata, exists := r.metadata[providerName]
	if !exists {
		// No metadata available, skip validation
		return nil
	}

	// Validate required options
	for _, required := range metadata.RequiredOptions {
		switch required {
		case "config_name":
			if options.ConfigName == "" {
				return NewError("validateProviderOptions", ErrCodeProviderConfigInvalid, providerName,
					fmt.Errorf("required option 'config_name' is empty"))
			}
		case "config_paths":
			if len(options.ConfigPaths) == 0 {
				return NewError("validateProviderOptions", ErrCodeProviderConfigInvalid, providerName,
					fmt.Errorf("required option 'config_paths' is empty"))
			}
		}
	}

	// Validate format support
	if options.Format != "" {
		supported := false
		for _, format := range metadata.SupportedFormats {
			if format == options.Format {
				supported = true
				break
			}
		}
		if !supported {
			return NewError("validateProviderOptions", ErrCodeFormatNotSupported, providerName,
				fmt.Errorf("format '%s' not supported by provider", options.Format))
		}
	}

	return nil
}

// getCreationTimeout returns the timeout for provider creation.
func (r *ConfigProviderRegistry) getCreationTimeout(providerName string) time.Duration {
	if metadata, exists := r.metadata[providerName]; exists && metadata.DefaultTimeout > 0 {
		return metadata.DefaultTimeout
	}
	return 30 * time.Second // Default timeout
}

// FindProviders returns providers matching the specified criteria.
func (r *ConfigProviderRegistry) FindProviders(criteria interface{}) ([]string, error) {
	// This is a simplified implementation - could be extended with more complex criteria
	r.mu.RLock()
	defer r.mu.RUnlock()

	var providers []string
	for name := range r.creators {
		providers = append(providers, name)
	}
	return providers, nil
}
