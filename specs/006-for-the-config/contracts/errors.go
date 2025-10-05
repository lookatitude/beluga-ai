// Package contracts defines the API contracts for Config package error handling.
// These interfaces provide structured error handling following the Op/Err/Code pattern
// required by the Beluga AI Framework constitution for consistent error management.
package contracts

import (
	"context"
	"time"
)

// ConfigError defines the interface for structured configuration errors.
// It implements the constitutional Op/Err/Code pattern for consistent error handling
// across the framework while preserving error chains and providing actionable information.
type ConfigError interface {
	error // Embed standard error interface

	// GetOperation returns the operation that failed.
	GetOperation() string

	// GetCode returns the standardized error code for programmatic handling.
	GetCode() string

	// GetProvider returns the provider name where the error occurred (if applicable).
	GetProvider() string

	// GetFormat returns the configuration format if applicable.
	GetFormat() string

	// GetConfigPath returns the configuration path if applicable.
	GetConfigPath() string

	// GetContext returns additional context information about the error.
	GetContext() map[string]interface{}

	// GetTimestamp returns when the error occurred.
	GetTimestamp() time.Time

	// IsRetryable returns whether the operation should be retried.
	IsRetryable() bool

	// GetRetryAfter returns the recommended delay before retry, if applicable.
	GetRetryAfter() time.Duration

	// Unwrap returns the underlying error for error chain preservation.
	Unwrap() error

	// Is supports errors.Is for error comparison.
	Is(target error) bool

	// As supports errors.As for type assertion.
	As(target interface{}) bool
}

// ErrorFactory defines the interface for creating structured configuration errors.
type ErrorFactory interface {
	// NewError creates a new ConfigError with the specified details.
	NewError(op, code, provider string, err error) ConfigError

	// NewLoadError creates a configuration loading error with format and path context.
	NewLoadError(op, code, provider, format, path string, err error) ConfigError

	// NewValidationError creates a validation error with detailed context.
	NewValidationError(op, code string, err error, validationContext map[string]interface{}) ConfigError

	// NewProviderError creates a provider-specific error.
	NewProviderError(op, code, provider string, err error) ConfigError

	// WrapError wraps an existing error with configuration-specific information.
	WrapError(err error, op, provider string) ConfigError
}

// Standard error codes for configuration operations following constitutional requirements.
const (
	// Registry Operations
	ErrCodeRegistryNotInitialized    = "REGISTRY_NOT_INITIALIZED"
	ErrCodeProviderAlreadyRegistered = "PROVIDER_ALREADY_REGISTERED"
	ErrCodeProviderNotFound          = "PROVIDER_NOT_FOUND"
	ErrCodeProviderCreationFailed    = "PROVIDER_CREATION_FAILED"
	ErrCodeInvalidProviderName       = "INVALID_PROVIDER_NAME"

	// Configuration Loading
	ErrCodeLoadFailed            = "LOAD_FAILED"
	ErrCodeFileNotFound          = "FILE_NOT_FOUND"
	ErrCodeFilePermissionDenied  = "FILE_PERMISSION_DENIED"
	ErrCodeParseError            = "PARSE_ERROR"
	ErrCodeFormatNotSupported    = "FORMAT_NOT_SUPPORTED"
	ErrCodeFormatDetectionFailed = "FORMAT_DETECTION_FAILED"

	// Validation Operations
	ErrCodeValidationFailed           = "VALIDATION_FAILED"
	ErrCodeSchemaValidationFailed     = "SCHEMA_VALIDATION_FAILED"
	ErrCodeRequiredFieldMissing       = "REQUIRED_FIELD_MISSING"
	ErrCodeInvalidFieldValue          = "INVALID_FIELD_VALUE"
	ErrCodeCrossFieldValidationFailed = "CROSS_FIELD_VALIDATION_FAILED"

	// Provider Configuration
	ErrCodeProviderConfigInvalid  = "PROVIDER_CONFIG_INVALID"
	ErrCodeProviderOptionsInvalid = "PROVIDER_OPTIONS_INVALID"
	ErrCodeProviderNotSupported   = "PROVIDER_NOT_SUPPORTED"
	ErrCodeProviderInitFailed     = "PROVIDER_INIT_FAILED"

	// Environment Variables
	ErrCodeEnvVarNotFound         = "ENV_VAR_NOT_FOUND"
	ErrCodeEnvVarParseError       = "ENV_VAR_PARSE_ERROR"
	ErrCodeEnvVarValidationFailed = "ENV_VAR_VALIDATION_FAILED"

	// Watch and Reload Operations
	ErrCodeWatchSetupFailed    = "WATCH_SETUP_FAILED"
	ErrCodeReloadFailed        = "RELOAD_FAILED"
	ErrCodeWatcherNotSupported = "WATCHER_NOT_SUPPORTED"

	// Health and Monitoring
	ErrCodeHealthCheckFailed      = "HEALTH_CHECK_FAILED"
	ErrCodeMetricsRecordingFailed = "METRICS_RECORDING_FAILED"
	ErrCodeTracingSetupFailed     = "TRACING_SETUP_FAILED"

	// Composite Provider Operations
	ErrCodeAllProvidersFailed   = "ALL_PROVIDERS_FAILED"
	ErrCodeFallbackFailed       = "FALLBACK_FAILED"
	ErrCodeCompositeSetupFailed = "COMPOSITE_SETUP_FAILED"

	// Internal Operations
	ErrCodeInternalError          = "INTERNAL_ERROR"
	ErrCodeConfigurationCorrupted = "CONFIGURATION_CORRUPTED"
	ErrCodeResourceExhausted      = "RESOURCE_EXHAUSTED"

	// Context and Timeout
	ErrCodeContextCancelled = "CONTEXT_CANCELLED"
	ErrCodeOperationTimeout = "OPERATION_TIMEOUT"
)

// ErrorClassifier defines the interface for classifying and analyzing errors.
type ErrorClassifier interface {
	// ClassifyError categorizes an error into standard classifications.
	ClassifyError(err error) ErrorClassification

	// IsRetryable determines if an error indicates a retryable condition.
	IsRetryable(err error) bool

	// GetRetryStrategy returns the recommended retry strategy for an error.
	GetRetryStrategy(err error) RetryStrategy

	// GetSeverityLevel returns the severity level of an error.
	GetSeverityLevel(err error) SeverityLevel

	// GetActionableAdvice returns human-readable advice for handling an error.
	GetActionableAdvice(err error) string
}

// ErrorClassification represents the classification of a configuration error.
type ErrorClassification struct {
	Category    ErrorCategory `json:"category"`
	Severity    SeverityLevel `json:"severity"`
	IsRetryable bool          `json:"is_retryable"`
	IsTemporary bool          `json:"is_temporary"`
	IsUserError bool          `json:"is_user_error"`
}

// ErrorCategory represents different categories of configuration errors.
type ErrorCategory string

const (
	CategoryRegistry    ErrorCategory = "registry"
	CategoryLoading     ErrorCategory = "loading"
	CategoryValidation  ErrorCategory = "validation"
	CategoryProvider    ErrorCategory = "provider"
	CategoryEnvironment ErrorCategory = "environment"
	CategoryFileSystem  ErrorCategory = "filesystem"
	CategoryParsing     ErrorCategory = "parsing"
	CategoryHealth      ErrorCategory = "health"
	CategoryInternal    ErrorCategory = "internal"
)

// SeverityLevel represents the severity of a configuration error.
type SeverityLevel string

const (
	SeverityInfo     SeverityLevel = "info"
	SeverityWarning  SeverityLevel = "warning"
	SeverityError    SeverityLevel = "error"
	SeverityCritical SeverityLevel = "critical"
)

// RetryStrategy provides guidance on how to handle retries for specific errors.
type RetryStrategy struct {
	ShouldRetry   bool          `json:"should_retry"`
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
	MaxDelay      time.Duration `json:"max_delay"`
	JitterEnabled bool          `json:"jitter_enabled"`
}

// ErrorReporter defines the interface for reporting and tracking configuration errors.
type ErrorReporter interface {
	// ReportError reports an error occurrence for tracking and analysis.
	ReportError(ctx context.Context, err ConfigError) error

	// GetErrorStats returns error statistics for analysis.
	GetErrorStats(timeWindow time.Duration) ErrorStats

	// GetTopErrors returns the most frequent errors in a time window.
	GetTopErrors(timeWindow time.Duration, limit int) []ErrorSummary

	// GetErrorsByProvider returns error breakdown by configuration provider.
	GetErrorsByProvider(timeWindow time.Duration) map[string]ErrorStats

	// GetErrorTrends returns error trends over time for monitoring.
	GetErrorTrends(timeWindow time.Duration, granularity time.Duration) []ErrorTrendPoint
}

// ErrorStats provides statistics about configuration error occurrences.
type ErrorStats struct {
	TotalErrors        int64                   `json:"total_errors"`
	ErrorRate          float64                 `json:"error_rate"` // errors per operation
	ErrorsByCode       map[string]int64        `json:"errors_by_code"`
	ErrorsByCategory   map[ErrorCategory]int64 `json:"errors_by_category"`
	ErrorsBySeverity   map[SeverityLevel]int64 `json:"errors_by_severity"`
	RetryableErrors    int64                   `json:"retryable_errors"`
	NonRetryableErrors int64                   `json:"non_retryable_errors"`
	TimeWindow         time.Duration           `json:"time_window"`
	LastUpdated        time.Time               `json:"last_updated"`
}

// ErrorSummary provides summary information about a specific error pattern.
type ErrorSummary struct {
	ErrorCode         string        `json:"error_code"`
	Category          ErrorCategory `json:"category"`
	Count             int64         `json:"count"`
	Rate              float64       `json:"rate"`
	FirstOccurrence   time.Time     `json:"first_occurrence"`
	LastOccurrence    time.Time     `json:"last_occurrence"`
	AffectedProviders []string      `json:"affected_providers"`
	AffectedFormats   []string      `json:"affected_formats"`
	SampleMessages    []string      `json:"sample_messages"`
	ActionableAdvice  string        `json:"actionable_advice"`
}

// ErrorTrendPoint represents error trend data at a specific time point.
type ErrorTrendPoint struct {
	Timestamp    time.Time        `json:"timestamp"`
	TotalErrors  int64            `json:"total_errors"`
	ErrorRate    float64          `json:"error_rate"`
	ErrorsByCode map[string]int64 `json:"errors_by_code"`
}

// ErrorHandler defines the interface for handling errors in a structured way.
type ErrorHandler interface {
	// HandleError processes an error and determines the appropriate action.
	HandleError(ctx context.Context, err error) ErrorAction

	// ShouldRetry determines if an operation should be retried based on the error.
	ShouldRetry(err error, attemptNumber int) bool

	// GetRetryDelay calculates the delay before the next retry attempt.
	GetRetryDelay(err error, attemptNumber int) time.Duration

	// LogError logs the error with appropriate detail level and structure.
	LogError(ctx context.Context, err error) error

	// NotifyError sends error notifications if configured.
	NotifyError(ctx context.Context, err error) error
}

// ErrorAction represents the recommended action to take when a configuration error occurs.
type ErrorAction struct {
	Action           ActionType    `json:"action"`
	ShouldRetry      bool          `json:"should_retry"`
	RetryDelay       time.Duration `json:"retry_delay,omitempty"`
	ShouldFallback   bool          `json:"should_fallback"`
	FallbackProvider string        `json:"fallback_provider,omitempty"`
	ShouldAlert      bool          `json:"should_alert"`
	AlertLevel       SeverityLevel `json:"alert_level,omitempty"`
	UserMessage      string        `json:"user_message,omitempty"`
}

// ActionType represents different types of actions for configuration errors.
type ActionType string

const (
	ActionRetry    ActionType = "retry"
	ActionFallback ActionType = "fallback"
	ActionFail     ActionType = "fail"
	ActionIgnore   ActionType = "ignore"
	ActionReload   ActionType = "reload"
)

// ErrorConfiguration defines configuration for error handling behavior.
type ErrorConfiguration struct {
	// Retry Configuration
	DefaultMaxRetries    int           `json:"default_max_retries"`
	DefaultInitialDelay  time.Duration `json:"default_initial_delay"`
	DefaultBackoffFactor float64       `json:"default_backoff_factor"`
	DefaultMaxDelay      time.Duration `json:"default_max_delay"`
	EnableJitter         bool          `json:"enable_jitter"`

	// Classification Configuration
	CustomClassifications map[string]ErrorClassification `json:"custom_classifications,omitempty"`
	SeverityThresholds    map[string]SeverityLevel       `json:"severity_thresholds,omitempty"`

	// Reporting Configuration
	EnableErrorReporting bool          `json:"enable_error_reporting"`
	ReportingInterval    time.Duration `json:"reporting_interval"`
	MaxReportedErrors    int           `json:"max_reported_errors"`

	// Notification Configuration
	EnableNotifications   bool                  `json:"enable_notifications"`
	NotificationThreshold map[SeverityLevel]int `json:"notification_threshold,omitempty"`
	NotificationChannels  []string              `json:"notification_channels,omitempty"`
}
