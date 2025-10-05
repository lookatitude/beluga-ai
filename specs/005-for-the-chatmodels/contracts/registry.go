// Package contracts defines the API contracts for ChatModels package registry operations.
// These interfaces enable global registry pattern compliance with provider management,
// discovery, and thread-safe operations.
package contracts

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

// ChatModelRegistry defines the interface for global chat model provider registry.
// It provides thread-safe provider registration, discovery, and creation capabilities.
type ChatModelRegistry interface {
	// RegisterGlobal registers a chat model provider creator function globally.
	// The creator function will be used to instantiate providers on demand.
	RegisterGlobal(name string, creator ProviderCreator) error

	// NewProvider creates a new chat model provider instance from a registered creator.
	// Returns error if provider is not registered or creation fails.
	NewProvider(ctx context.Context, name string, config Config) (iface.ChatModel, error)

	// ListProviders returns a list of all registered provider names.
	ListProviders() []string

	// GetProviderMetadata returns metadata for a registered provider.
	// Includes capabilities, required configuration, and supported models.
	GetProviderMetadata(name string) (*ProviderMetadata, error)

	// IsProviderRegistered checks if a provider with the given name is registered.
	IsProviderRegistered(name string) bool

	// UnregisterProvider removes a provider from the registry.
	// Returns error if provider is not registered or cannot be removed.
	UnregisterProvider(name string) error
}

// ProviderCreator defines the function signature for creating chat model providers.
// It takes a context and configuration, returning a ChatModel implementation or error.
type ProviderCreator func(ctx context.Context, config Config) (iface.ChatModel, error)

// ProviderMetadata contains information about a chat model provider's capabilities
// and configuration requirements.
type ProviderMetadata struct {
	// Basic Information
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Version     string `json:"version"`

	// Capabilities
	Capabilities    []string `json:"capabilities" validate:"required"`
	SupportedModels []string `json:"supported_models" validate:"required"`

	// Configuration Requirements
	RequiredConfig []string `json:"required_config"`
	OptionalConfig []string `json:"optional_config"`

	// Operational Metadata
	HealthCheckURL    string        `json:"health_check_url,omitempty"`
	DefaultTimeout    time.Duration `json:"default_timeout"`
	MaxRetries        int           `json:"max_retries"`
	SupportsStreaming bool          `json:"supports_streaming"`
	SupportsToolCalls bool          `json:"supports_tool_calls"`
	SupportsBatching  bool          `json:"supports_batching"`

	// Provider-specific Information
	APIVersion      string             `json:"api_version,omitempty"`
	Endpoints       map[string]string  `json:"endpoints,omitempty"`
	RateLimits      map[string]int     `json:"rate_limits,omitempty"`
	CostInformation map[string]float64 `json:"cost_information,omitempty"`
}

// Config represents the configuration for creating a chat model provider.
// It combines common configuration with provider-specific options.
type Config struct {
	// Provider Selection
	Provider string `json:"provider" validate:"required"`
	Model    string `json:"model" validate:"required"`

	// Authentication
	APIKey      string `json:"api_key,omitempty"`
	AccessToken string `json:"access_token,omitempty"`

	// Connection Settings
	BaseURL        string        `json:"base_url,omitempty"`
	Timeout        time.Duration `json:"timeout" validate:"min=1s"`
	MaxRetries     int           `json:"max_retries" validate:"min=0,max=10"`
	RequestsPerMin int           `json:"requests_per_min" validate:"min=1"`

	// Model Parameters
	Temperature   float32  `json:"temperature" validate:"min=0,max=2"`
	MaxTokens     int      `json:"max_tokens" validate:"min=1"`
	TopP          float32  `json:"top_p" validate:"min=0,max=1"`
	StopSequences []string `json:"stop_sequences,omitempty"`

	// Features
	EnableStreaming bool `json:"enable_streaming"`
	EnableToolCalls bool `json:"enable_tool_calls"`
	EnableBatching  bool `json:"enable_batching"`

	// Observability
	EnableMetrics bool `json:"enable_metrics"`
	EnableTracing bool `json:"enable_tracing"`
	EnableLogging bool `json:"enable_logging"`

	// Provider-Specific Options
	ProviderSpecific map[string]interface{} `json:"provider_specific,omitempty"`
}

// RegistryValidator defines the interface for validating registry operations
// and configurations before provider creation.
type RegistryValidator interface {
	// ValidateProviderName checks if a provider name is valid for registration.
	ValidateProviderName(name string) error

	// ValidateConfig validates a configuration against provider requirements.
	ValidateConfig(providerName string, config Config) error

	// ValidateMetadata validates provider metadata for completeness and correctness.
	ValidateMetadata(metadata ProviderMetadata) error

	// ValidateCapability checks if a capability string is recognized.
	ValidateCapability(capability string) error
}

// ProviderDiscovery defines the interface for discovering and querying
// available chat model providers and their capabilities.
type ProviderDiscovery interface {
	// FindProviders returns providers matching the specified criteria.
	FindProviders(criteria DiscoveryCriteria) ([]ProviderMetadata, error)

	// GetProvidersWithCapability returns providers supporting a specific capability.
	GetProvidersWithCapability(capability string) ([]string, error)

	// GetProvidersForModel returns providers that support a specific model.
	GetProvidersForModel(modelName string) ([]string, error)

	// GetBestProvider recommends the best provider for given requirements.
	GetBestProvider(requirements ProviderRequirements) (string, error)
}

// DiscoveryCriteria specifies criteria for provider discovery operations.
type DiscoveryCriteria struct {
	Capabilities    []string           `json:"capabilities,omitempty"`
	ModelSupport    []string           `json:"model_support,omitempty"`
	MaxLatency      time.Duration      `json:"max_latency,omitempty"`
	MinReliability  float64            `json:"min_reliability,omitempty"`
	CostConstraints map[string]float64 `json:"cost_constraints,omitempty"`
}

// ProviderRequirements specifies requirements for provider recommendation.
type ProviderRequirements struct {
	// Functional Requirements
	RequiredCapabilities []string `json:"required_capabilities"`
	PreferredModels      []string `json:"preferred_models"`

	// Performance Requirements
	MaxLatency       time.Duration `json:"max_latency,omitempty"`
	MinThroughput    int           `json:"min_throughput,omitempty"`
	ReliabilityLevel float64       `json:"reliability_level,omitempty"`

	// Cost Requirements
	MaxCostPerToken  float64 `json:"max_cost_per_token,omitempty"`
	MaxMonthlyCost   float64 `json:"max_monthly_cost,omitempty"`
	CostOptimization bool    `json:"cost_optimization"`

	// Operational Requirements
	RequireStreaming   bool `json:"require_streaming"`
	RequireToolCalls   bool `json:"require_tool_calls"`
	RequireBatching    bool `json:"require_batching"`
	RequireHealthCheck bool `json:"require_health_check"`
}

// RegistryConfiguration defines configuration options for the registry itself.
type RegistryConfiguration struct {
	// Concurrency Settings
	MaxConcurrentCreations int           `json:"max_concurrent_creations"`
	CreationTimeout        time.Duration `json:"creation_timeout"`

	// Caching Settings
	EnableProviderCaching bool          `json:"enable_provider_caching"`
	CacheTTL              time.Duration `json:"cache_ttl"`
	MaxCacheSize          int           `json:"max_cache_size"`

	// Health Checking
	EnableHealthChecks   bool          `json:"enable_health_checks"`
	HealthCheckInterval  time.Duration `json:"health_check_interval"`
	HealthCheckTimeout   time.Duration `json:"health_check_timeout"`
	MaxFailedHealthCheck int           `json:"max_failed_health_checks"`

	// Observability
	EnableRegistryMetrics bool   `json:"enable_registry_metrics"`
	MetricsPrefix         string `json:"metrics_prefix"`
}

// RegistryFactory defines the interface for creating and configuring registries.
type RegistryFactory interface {
	// CreateRegistry creates a new chat model registry with the specified configuration.
	CreateRegistry(config RegistryConfiguration) (ChatModelRegistry, error)

	// CreateDefaultRegistry creates a registry with default configuration.
	CreateDefaultRegistry() (ChatModelRegistry, error)

	// GetGlobalRegistry returns the global singleton registry instance.
	GetGlobalRegistry() ChatModelRegistry
}

// RegistryHealth defines the interface for registry health monitoring.
type RegistryHealth interface {
	// CheckHealth returns the overall health status of the registry.
	CheckHealth() RegistryHealthStatus

	// CheckProviderHealth returns health status for a specific provider.
	CheckProviderHealth(providerName string) ProviderHealthStatus

	// GetHealthSummary returns a summary of all provider health statuses.
	GetHealthSummary() map[string]ProviderHealthStatus
}

// RegistryHealthStatus represents the overall health of the registry.
type RegistryHealthStatus struct {
	Status           string                          `json:"status"`
	RegisteredCount  int                             `json:"registered_count"`
	HealthyCount     int                             `json:"healthy_count"`
	UnhealthyCount   int                             `json:"unhealthy_count"`
	LastChecked      time.Time                       `json:"last_checked"`
	ProviderStatuses map[string]ProviderHealthStatus `json:"provider_statuses"`
}

// ProviderHealthStatus represents the health status of a specific provider.
type ProviderHealthStatus struct {
	Status         string            `json:"status"`
	LastChecked    time.Time         `json:"last_checked"`
	ResponseTime   time.Duration     `json:"response_time,omitempty"`
	ErrorCount     int               `json:"error_count"`
	SuccessRate    float64           `json:"success_rate"`
	LastError      string            `json:"last_error,omitempty"`
	AdditionalInfo map[string]string `json:"additional_info,omitempty"`
}
