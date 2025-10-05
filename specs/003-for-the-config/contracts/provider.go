// Package contracts defines the API contracts for provider operations
// These interfaces define the contract for configuration provider functionality
// implementing FR-003, FR-004, FR-017, FR-018, FR-021, FR-022

package contracts

import (
	"context"
	"time"
)

// Provider defines the core contract for configuration providers
type Provider interface {
	// Core loading operations
	// Implements FR-017: System MUST support provider-specific configuration
	Load(ctx context.Context, configStruct interface{}) error
	UnmarshalKey(ctx context.Context, key string, rawVal interface{}) error

	// Value retrieval operations
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetFloat64(key string) float64
	GetStringMapString(key string) map[string]string
	IsSet(key string) bool

	// Validation and defaults
	// Implements FR-017: System MUST support provider-specific configuration with validation
	Validate(ctx context.Context) error
	SetDefaults(ctx context.Context) error

	// Lifecycle management
	// Implements FR-021: System MUST respect context cancellation
	Initialize(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// EnhancedProvider extends Provider with advanced capabilities
type EnhancedProvider interface {
	Provider
	HealthChecker
	HotReloader
	MetricsReporter
	RetryableProvider
}

// HotReloader defines the contract for hot-reloading capabilities
type HotReloader interface {
	// EnableHotReload starts watching for configuration changes
	// Implements FR-004: System MUST provide configuration hot-reload capabilities
	EnableHotReload(ctx context.Context, callback ReloadCallback) error

	// DisableHotReload stops watching for configuration changes
	DisableHotReload(ctx context.Context) error

	// GetReloadStatus returns current hot-reload status
	GetReloadStatus() ReloadStatus
}

// MetricsReporter defines the contract for provider metrics
type MetricsReporter interface {
	// GetMetrics returns current provider metrics
	// Implements FR-012: System MUST collect and expose metrics
	GetMetrics() ProviderMetrics

	// ResetMetrics resets all metrics counters
	ResetMetrics() error
}

// RetryableProvider defines retry capabilities
type RetryableProvider interface {
	// RetryOperation executes an operation with retry logic
	// Implements FR-022: System MUST implement retry logic with exponential backoff
	RetryOperation(ctx context.Context, operation RetryableOperation, options RetryOptions) error

	// SetRetryPolicy configures the retry policy for this provider
	SetRetryPolicy(policy RetryPolicy) error
}

// CompositeProvider defines the contract for provider composition
type CompositeProvider interface {
	Provider

	// AddProvider adds a provider to the composition chain
	// Implements FR-018: System MUST enable provider composition
	AddProvider(priority int, provider Provider) error

	// RemoveProvider removes a provider from the composition chain
	RemoveProvider(provider Provider) error

	// GetProviders returns all providers in the composition chain
	GetProviders() []Provider

	// SetFallbackPolicy configures how providers are used as fallbacks
	// Implements FR-020: System MUST provide graceful degradation when providers fail
	SetFallbackPolicy(policy FallbackPolicy) error
}

// ReloadCallback defines the signature for hot-reload callbacks
type ReloadCallback func(oldConfig, newConfig interface{}) error

// ReloadStatus represents the current hot-reload state
type ReloadStatus struct {
	Enabled       bool      `json:"enabled"`
	WatchingPaths []string  `json:"watching_paths"`
	LastReload    time.Time `json:"last_reload"`
	ReloadCount   int64     `json:"reload_count"`
	ErrorCount    int64     `json:"error_count"`
}

// ProviderMetrics contains metrics for a provider
type ProviderMetrics struct {
	LoadCount       int64         `json:"load_count"`
	LoadDuration    time.Duration `json:"load_duration"`
	ErrorCount      int64         `json:"error_count"`
	LastHealthCheck time.Time     `json:"last_health_check"`
	HealthStatus    string        `json:"health_status"`
	RetryCount      int64         `json:"retry_count"`
	CacheHits       int64         `json:"cache_hits"`
	CacheMisses     int64         `json:"cache_misses"`
}

// RetryableOperation defines an operation that can be retried
type RetryableOperation func(ctx context.Context) error

// RetryOptions configures retry behavior for specific operations
type RetryOptions struct {
	MaxAttempts   int           `mapstructure:"max_attempts" default:"3"`
	InitialDelay  time.Duration `mapstructure:"initial_delay" default:"100ms"`
	MaxDelay      time.Duration `mapstructure:"max_delay" default:"30s"`
	Multiplier    float64       `mapstructure:"multiplier" default:"2.0"`
	Jitter        bool          `mapstructure:"jitter" default:"true"`
	RetryOnErrors []string      `mapstructure:"retry_on_errors"`
}

// RetryPolicy defines the overall retry policy for a provider
type RetryPolicy struct {
	DefaultOptions   RetryOptions            `mapstructure:"default"`
	OperationOptions map[string]RetryOptions `mapstructure:"operations"`
	Enabled          bool                    `mapstructure:"enabled" default:"true"`
}

// FallbackPolicy defines how providers should fallback when others fail
type FallbackPolicy struct {
	Strategy             FallbackStrategy `mapstructure:"strategy" default:"priority"`
	MaxFailures          int              `mapstructure:"max_failures" default:"3"`
	FailureWindow        time.Duration    `mapstructure:"failure_window" default:"5m"`
	RecoveryDelay        time.Duration    `mapstructure:"recovery_delay" default:"30s"`
	EnableCircuitBreaker bool             `mapstructure:"enable_circuit_breaker" default:"true"`
}

// FallbackStrategy defines different fallback approaches
type FallbackStrategy string

const (
	FallbackStrategyPriority   FallbackStrategy = "priority"    // Try providers in priority order
	FallbackStrategyRoundRobin FallbackStrategy = "round_robin" // Rotate through healthy providers
	FallbackStrategyFailFast   FallbackStrategy = "fail_fast"   // Fail immediately on error
	FallbackStrategyBestEffort FallbackStrategy = "best_effort" // Try all providers, return partial results
)

// ProviderError defines structured errors for provider operations
type ProviderError struct {
	Op       string // operation that failed
	Provider string // provider name
	Key      string // configuration key involved
	Err      error  // underlying error
	Code     string // error code
}

const (
	ErrCodeLoadFailed       = "LOAD_FAILED"
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	ErrCodeKeyNotFound      = "KEY_NOT_FOUND"
	ErrCodeReloadFailed     = "RELOAD_FAILED"
	ErrCodeRetryExhausted   = "RETRY_EXHAUSTED"
	ErrCodeContextCanceled  = "CONTEXT_CANCELED"
	ErrCodeTimeout          = "TIMEOUT"
)
