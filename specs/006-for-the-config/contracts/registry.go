// Package contracts defines the API contracts for Config package provider registry operations.
// These interfaces enable global registry pattern compliance with provider management,
// discovery, and thread-safe operations for configuration providers.
package contracts

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
)

// ConfigProviderRegistry defines the interface for global configuration provider registry.
// It provides thread-safe provider registration, discovery, and creation capabilities.
type ConfigProviderRegistry interface {
	// RegisterGlobal registers a configuration provider creator function globally.
	// The creator function will be used to instantiate providers on demand.
	RegisterGlobal(name string, creator ProviderCreator) error

	// NewProvider creates a new configuration provider instance from a registered creator.
	// Returns error if provider is not registered or creation fails.
	NewProvider(ctx context.Context, name string, options ProviderOptions) (iface.Provider, error)

	// ListProviders returns a list of all registered provider names.
	ListProviders() []string

	// GetProviderMetadata returns metadata for a registered provider.
	// Includes capabilities, supported formats, and configuration requirements.
	GetProviderMetadata(name string) (*ProviderMetadata, error)

	// IsProviderRegistered checks if a provider with the given name is registered.
	IsProviderRegistered(name string) bool

	// UnregisterProvider removes a provider from the registry.
	// Returns error if provider is not registered or cannot be removed safely.
	UnregisterProvider(name string) error

	// GetProvidersForFormat returns providers that support a specific configuration format.
	GetProvidersForFormat(format string) ([]string, error)

	// GetProviderByCapability returns providers that support a specific capability.
	GetProviderByCapability(capability string) ([]string, error)
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

// RegistryValidator defines the interface for validating registry operations.
type RegistryValidator interface {
	// ValidateProviderName checks if a provider name is valid for registration.
	ValidateProviderName(name string) error

	// ValidateProviderOptions validates provider options against provider requirements.
	ValidateProviderOptions(providerName string, options ProviderOptions) error

	// ValidateProviderMetadata validates provider metadata for completeness.
	ValidateProviderMetadata(metadata ProviderMetadata) error

	// ValidateProviderCapability checks if a capability string is recognized.
	ValidateProviderCapability(capability string) error
}

// ProviderDiscovery defines the interface for discovering and querying available providers.
type ProviderDiscovery interface {
	// FindProviders returns providers matching the specified criteria.
	FindProviders(criteria DiscoveryCriteria) ([]ProviderMetadata, error)

	// GetProvidersForFormat returns providers supporting a specific configuration format.
	GetProvidersForFormat(format string) ([]string, error)

	// GetProvidersWithCapability returns providers supporting a specific capability.
	GetProvidersWithCapability(capability string) ([]string, error)

	// GetBestProvider recommends the best provider for given requirements.
	GetBestProvider(requirements ProviderRequirements) (string, error)
}

// DiscoveryCriteria specifies criteria for provider discovery operations.
type DiscoveryCriteria struct {
	RequiredFormats      []string               `json:"required_formats,omitempty"`
	RequiredCapabilities []string               `json:"required_capabilities,omitempty"`
	MaxLoadTime          time.Duration          `json:"max_load_time,omitempty"`
	RequireHealthCheck   bool                   `json:"require_health_check"`
	RequireCaching       bool                   `json:"require_caching"`
	CustomFilters        map[string]interface{} `json:"custom_filters,omitempty"`
}

// ProviderRequirements specifies requirements for provider recommendation.
type ProviderRequirements struct {
	// Format Requirements
	PreferredFormats []string `json:"preferred_formats"`
	RequiredFormats  []string `json:"required_formats"`

	// Capability Requirements
	RequiredCapabilities  []string `json:"required_capabilities"`
	PreferredCapabilities []string `json:"preferred_capabilities"`

	// Performance Requirements
	MaxLoadTime     time.Duration `json:"max_load_time,omitempty"`
	RequireWatching bool          `json:"require_watching"`
	RequireCaching  bool          `json:"require_caching"`

	// Operational Requirements
	RequireHealthCheck bool `json:"require_health_check"`
	RequireValidation  bool `json:"require_validation"`
	RequireEnvSupport  bool `json:"require_env_support"`
}

// RegistryConfiguration defines configuration options for the registry itself.
type RegistryConfiguration struct {
	// Concurrency Settings
	MaxConcurrentCreations int           `json:"max_concurrent_creations"`
	CreationTimeout        time.Duration `json:"creation_timeout"`

	// Caching Settings
	EnableProviderCaching bool          `json:"enable_provider_caching"`
	ProviderCacheTTL      time.Duration `json:"provider_cache_ttl"`
	MaxCacheSize          int           `json:"max_cache_size"`

	// Health Monitoring
	EnableHealthChecks  bool          `json:"enable_health_checks"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`

	// Observability
	EnableRegistryMetrics bool   `json:"enable_registry_metrics"`
	MetricsPrefix         string `json:"metrics_prefix"`
}

// RegistryFactory defines the interface for creating and configuring provider registries.
type RegistryFactory interface {
	// CreateRegistry creates a new configuration provider registry with specified configuration.
	CreateRegistry(config RegistryConfiguration) (ConfigProviderRegistry, error)

	// CreateDefaultRegistry creates a registry with default configuration suitable for most use cases.
	CreateDefaultRegistry() (ConfigProviderRegistry, error)

	// GetGlobalRegistry returns the global singleton registry instance.
	GetGlobalRegistry() ConfigProviderRegistry
}

// RegistryHealth defines the interface for registry health monitoring.
type RegistryHealth interface {
	// CheckHealth returns the overall health status of the registry and its providers.
	CheckHealth() RegistryHealthStatus

	// CheckProviderHealth returns health status for a specific registered provider.
	CheckProviderHealth(providerName string) ProviderHealthStatus

	// GetHealthSummary returns a summary of all provider health statuses.
	GetHealthSummary() map[string]ProviderHealthStatus

	// StartHealthMonitoring begins periodic health checks for all providers.
	StartHealthMonitoring(interval time.Duration) error

	// StopHealthMonitoring stops periodic health checking.
	StopHealthMonitoring() error
}

// RegistryHealthStatus represents the overall health of the configuration registry.
type RegistryHealthStatus struct {
	Status           string                          `json:"status"`
	RegisteredCount  int                             `json:"registered_count"`
	HealthyCount     int                             `json:"healthy_count"`
	UnhealthyCount   int                             `json:"unhealthy_count"`
	DegradedCount    int                             `json:"degraded_count"`
	LastChecked      time.Time                       `json:"last_checked"`
	ProviderStatuses map[string]ProviderHealthStatus `json:"provider_statuses"`
	OverallScore     float64                         `json:"overall_score"`
}

// ProviderHealthStatus represents the health status of a specific provider.
type ProviderHealthStatus struct {
	Status           string            `json:"status"`
	LastChecked      time.Time         `json:"last_checked"`
	ResponseTime     time.Duration     `json:"response_time,omitempty"`
	ErrorCount       int               `json:"error_count"`
	SuccessRate      float64           `json:"success_rate"`
	LastError        string            `json:"last_error,omitempty"`
	LoadCount        int64             `json:"load_count"`
	Capabilities     []string          `json:"capabilities"`
	SupportedFormats []string          `json:"supported_formats"`
	AdditionalInfo   map[string]string `json:"additional_info,omitempty"`
}

// ConfigLoaderEnhanced defines the enhanced interface for configuration loading with health monitoring.
type ConfigLoaderEnhanced interface {
	iface.Loader

	// LoadWithHealth loads configuration with health monitoring and performance tracking.
	LoadWithHealth(ctx context.Context) (*iface.Config, LoadResult, error)

	// ReloadConfig reloads configuration with change detection and validation.
	ReloadConfig(ctx context.Context) (*iface.Config, ReloadResult, error)

	// GetLoaderHealth returns current health status of the loader and its providers.
	GetLoaderHealth() LoaderHealthStatus
}

// LoadResult contains detailed information about a configuration load operation.
type LoadResult struct {
	Duration       time.Duration          `json:"duration"`
	ProviderUsed   string                 `json:"provider_used"`
	FormatDetected string                 `json:"format_detected"`
	SourcesParsed  []string               `json:"sources_parsed"`
	ValidationTime time.Duration          `json:"validation_time"`
	ErrorCount     int                    `json:"error_count"`
	WarningCount   int                    `json:"warning_count"`
	CacheHit       bool                   `json:"cache_hit"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ReloadResult contains information about a configuration reload operation.
type ReloadResult struct {
	LoadResult
	ChangesDetected bool     `json:"changes_detected"`
	ChangedKeys     []string `json:"changed_keys,omitempty"`
	PreviousHash    string   `json:"previous_hash"`
	CurrentHash     string   `json:"current_hash"`
}

// LoaderHealthStatus represents the health status of a configuration loader.
type LoaderHealthStatus struct {
	Status             string                 `json:"status"`
	LoaderReady        bool                   `json:"loader_ready"`
	ProvidersHealthy   int                    `json:"providers_healthy"`
	ProvidersTotal     int                    `json:"providers_total"`
	LastSuccessfulLoad time.Time              `json:"last_successful_load"`
	LoadErrorRate      float64                `json:"load_error_rate"`
	AverageLoadTime    time.Duration          `json:"average_load_time"`
	ValidationStatus   ValidationHealthStatus `json:"validation_status"`
}

// ValidationHealthStatus represents the health of the validation system.
type ValidationHealthStatus struct {
	Status                string        `json:"status"`
	ValidationsPerformed  int64         `json:"validations_performed"`
	ValidationErrors      int64         `json:"validation_errors"`
	SuccessRate           float64       `json:"success_rate"`
	AverageValidationTime time.Duration `json:"average_validation_time"`
	LastValidation        time.Time     `json:"last_validation"`
}
