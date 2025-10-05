// Package contracts defines the API contracts for the config package registry operations
// These interfaces define the contract for provider registry functionality
// implementing FR-001, FR-015, FR-016

package contracts

import (
	"context"
	"time"
)

// ProviderRegistry defines the contract for centralized provider management
type ProviderRegistry interface {
	// RegisterProvider registers a new provider creator function
	// Implements FR-001: System MUST provide a centralized provider registry
	RegisterProvider(name string, creator ProviderCreatorFunc) error

	// UnregisterProvider removes a provider from the registry
	// Implements FR-016: System MUST provide provider lifecycle management
	UnregisterProvider(name string) error

	// CreateProvider creates a new provider instance using registered creator
	// Implements FR-015: System MUST support dynamic provider registration at runtime
	CreateProvider(ctx context.Context, name string, config interface{}) (Provider, error)

	// ListProviders returns all registered provider names
	// Supports provider discovery functionality
	ListProviders() []string

	// GetProviderHealth returns the health status of a specific provider
	// Implements FR-002: System MUST implement comprehensive health checks
	GetProviderHealth(name string) HealthStatus

	// GetSystemHealth returns overall system health status
	// Aggregates health across all providers
	GetSystemHealth(ctx context.Context) SystemHealth

	// Shutdown gracefully shuts down all providers and cleans up resources
	// Implements FR-016: System MUST provide provider lifecycle management
	Shutdown(ctx context.Context) error

	// GetMetrics returns registry operation metrics
	// Implements FR-012: System MUST collect and expose metrics
	GetMetrics() RegistryMetrics
}

// ProviderCreatorFunc defines the contract for provider creation functions
type ProviderCreatorFunc func(ctx context.Context, config interface{}) (Provider, error)

// RegistryMetrics defines metrics exposed by the registry
type RegistryMetrics struct {
	RegisteredProviders int           `json:"registered_providers"`
	ActiveProviders     int           `json:"active_providers"`
	HealthyProviders    int           `json:"healthy_providers"`
	RegistrationCount   int64         `json:"registration_count"`
	CreationCount       int64         `json:"creation_count"`
	FailureCount        int64         `json:"failure_count"`
	LastHealthCheck     time.Time     `json:"last_health_check"`
	AverageCreationTime time.Duration `json:"average_creation_time"`
}

// RegistryOptions defines configuration options for the registry
type RegistryOptions struct {
	MaxProviders        int           `mapstructure:"max_providers" default:"100"`
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval" default:"30s"`
	HealthCheckTimeout  time.Duration `mapstructure:"health_check_timeout" default:"5s"`
	EnableMetrics       bool          `mapstructure:"enable_metrics" default:"true"`
	EnableTracing       bool          `mapstructure:"enable_tracing" default:"true"`
	ShutdownTimeout     time.Duration `mapstructure:"shutdown_timeout" default:"30s"`
}

// RegistryError defines structured errors for registry operations
type RegistryError struct {
	Op       string // operation that failed
	Provider string // provider name involved
	Err      error  // underlying error
	Code     string // error code
}

const (
	ErrCodeProviderExists    = "PROVIDER_EXISTS"
	ErrCodeProviderNotFound  = "PROVIDER_NOT_FOUND"
	ErrCodeCreationFailed    = "CREATION_FAILED"
	ErrCodeShutdownFailed    = "SHUTDOWN_FAILED"
	ErrCodeHealthCheckFailed = "HEALTH_CHECK_FAILED"
	ErrCodeRegistryFull      = "REGISTRY_FULL"
)
