// Package contracts defines the API contracts for ChatModels package error handling.
// These interfaces provide structured error handling following the Op/Err/Code pattern
// required by the Beluga AI Framework constitution for consistent error management.
package contracts

import (
	"context"
	"time"
)

// ChatModelError defines the interface for structured chat model errors.
// It implements the constitutional Op/Err/Code pattern for consistent error handling
// across the framework while preserving error chains and providing actionable information.
type ChatModelError interface {
	error // Embed standard error interface

	// GetOperation returns the operation that failed.
	GetOperation() string

	// GetCode returns the standardized error code for programmatic handling.
	GetCode() string

	// GetProvider returns the provider name where the error occurred.
	GetProvider() string

	// GetModel returns the model identifier if applicable.
	GetModel() string

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

// ErrorFactory defines the interface for creating structured errors.
// It provides consistent error creation with proper context and classification.
type ErrorFactory interface {
	// NewError creates a new ChatModelError with the specified details.
	NewError(op, code, provider string, err error) ChatModelError

	// NewContextError creates a new error with additional context information.
	NewContextError(op, code, provider string, err error, context map[string]interface{}) ChatModelError

	// NewProviderError creates a provider-specific error with model information.
	NewProviderError(op, code, provider, model string, err error) ChatModelError

	// WrapError wraps an existing error with ChatModel-specific information.
	WrapError(err error, op, provider string) ChatModelError

	// FromStandardError converts a standard error to a ChatModelError.
	FromStandardError(err error, op, provider string) ChatModelError
}

// ErrorClassifier defines the interface for classifying and analyzing errors.
// It provides error categorization and actionable recommendations.
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

// ErrorReporter defines the interface for reporting and tracking errors.
// It provides error aggregation, analysis, and reporting capabilities.
type ErrorReporter interface {
	// ReportError reports an error occurrence for tracking and analysis.
	ReportError(ctx context.Context, err ChatModelError) error

	// GetErrorStats returns error statistics for analysis.
	GetErrorStats(timeWindow time.Duration) ErrorStats

	// GetTopErrors returns the most frequent errors in a time window.
	GetTopErrors(timeWindow time.Duration, limit int) []ErrorSummary

	// GetErrorsByProvider returns error breakdown by provider.
	GetErrorsByProvider(timeWindow time.Duration) map[string]ErrorStats

	// GetErrorTrends returns error trends over time for monitoring.
	GetErrorTrends(timeWindow time.Duration, granularity time.Duration) []ErrorTrendPoint
}

// Standard error codes following constitutional requirements.
// These codes enable programmatic error handling across providers.
const (
	// Configuration Errors
	ErrCodeConfigInvalid       = "CONFIG_INVALID"
	ErrCodeConfigMissing       = "CONFIG_MISSING"
	ErrCodeProviderUnsupported = "PROVIDER_UNSUPPORTED"
	ErrCodeModelUnsupported    = "MODEL_UNSUPPORTED"

	// Authentication Errors
	ErrCodeAuthenticationFailed = "AUTHENTICATION_FAILED"
	ErrCodeAPIKeyInvalid        = "API_KEY_INVALID"
	ErrCodeAPIKeyMissing        = "API_KEY_MISSING"
	ErrCodePermissionDenied     = "PERMISSION_DENIED"

	// Request Errors
	ErrCodeInvalidInput         = "INVALID_INPUT"
	ErrCodeRequestTooLarge      = "REQUEST_TOO_LARGE"
	ErrCodeUnsupportedOperation = "UNSUPPORTED_OPERATION"
	ErrCodeMalformedRequest     = "MALFORMED_REQUEST"

	// Service Errors
	ErrCodeProviderUnavailable = "PROVIDER_UNAVAILABLE"
	ErrCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	ErrCodeRequestTimeout      = "REQUEST_TIMEOUT"
	ErrCodeConnectionFailed    = "CONNECTION_FAILED"

	// Rate Limiting Errors
	ErrCodeRateLimit        = "RATE_LIMIT"
	ErrCodeQuotaExceeded    = "QUOTA_EXCEEDED"
	ErrCodeConcurrencyLimit = "CONCURRENCY_LIMIT"

	// Processing Errors
	ErrCodeProcessingFailed   = "PROCESSING_FAILED"
	ErrCodeModelOverloaded    = "MODEL_OVERLOADED"
	ErrCodeContentFiltered    = "CONTENT_FILTERED"
	ErrCodeTokenLimitExceeded = "TOKEN_LIMIT_EXCEEDED"

	// Internal Errors
	ErrCodeInternalError  = "INTERNAL_ERROR"
	ErrCodeUnknownError   = "UNKNOWN_ERROR"
	ErrCodeNotImplemented = "NOT_IMPLEMENTED"

	// Context Errors
	ErrCodeContextCancelled = "CONTEXT_CANCELLED"
	ErrCodeContextTimeout   = "CONTEXT_TIMEOUT"
)

// ErrorClassification represents the classification of an error.
type ErrorClassification struct {
	Category    ErrorCategory `json:"category"`
	Severity    SeverityLevel `json:"severity"`
	IsRetryable bool          `json:"is_retryable"`
	IsTemporary bool          `json:"is_temporary"`
	IsUserError bool          `json:"is_user_error"`
}

// ErrorCategory represents different categories of errors.
type ErrorCategory string

const (
	CategoryConfiguration  ErrorCategory = "configuration"
	CategoryAuthentication ErrorCategory = "authentication"
	CategoryRequest        ErrorCategory = "request"
	CategoryService        ErrorCategory = "service"
	CategoryRateLimit      ErrorCategory = "rate_limit"
	CategoryProcessing     ErrorCategory = "processing"
	CategoryInternal       ErrorCategory = "internal"
	CategoryContext        ErrorCategory = "context"
)

// SeverityLevel represents the severity of an error.
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
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

// ErrorStats provides statistics about error occurrences.
type ErrorStats struct {
	TotalErrors           int64                   `json:"total_errors"`
	ErrorRate             float64                 `json:"error_rate"` // errors per request
	ErrorsByCode          map[string]int64        `json:"errors_by_code"`
	ErrorsByCategory      map[ErrorCategory]int64 `json:"errors_by_category"`
	ErrorsBySeverity      map[SeverityLevel]int64 `json:"errors_by_severity"`
	RetryableErrors       int64                   `json:"retryable_errors"`
	NonRetryableErrors    int64                   `json:"non_retryable_errors"`
	AverageResolutionTime time.Duration           `json:"average_resolution_time"`
	TimeWindow            time.Duration           `json:"time_window"`
	LastUpdated           time.Time               `json:"last_updated"`
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

// ContextualErrorInfo provides additional context information for errors.
type ContextualErrorInfo struct {
	// Request Context
	RequestID   string `json:"request_id,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`

	// Technical Context
	ProviderName string `json:"provider_name"`
	ModelName    string `json:"model_name,omitempty"`
	APIVersion   string `json:"api_version,omitempty"`
	Endpoint     string `json:"endpoint,omitempty"`

	// Error Context
	ErrorCode    string        `json:"error_code"`
	HttpStatus   int           `json:"http_status,omitempty"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
	RetryCount   int           `json:"retry_count"`

	// Additional Context
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	StackTrace   string                 `json:"stack_trace,omitempty"`
	RequestBody  string                 `json:"request_body,omitempty"`  // Sanitized
	ResponseBody string                 `json:"response_body,omitempty"` // Sanitized
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

	// NotifyError sends error notifications if configured (alerts, webhooks, etc.).
	NotifyError(ctx context.Context, err error) error
}

// ErrorAction represents the recommended action to take when an error occurs.
type ErrorAction struct {
	Action            ActionType    `json:"action"`
	ShouldRetry       bool          `json:"should_retry"`
	RetryDelay        time.Duration `json:"retry_delay,omitempty"`
	ShouldFallback    bool          `json:"should_fallback"`
	FallbackProvider  string        `json:"fallback_provider,omitempty"`
	ShouldNotify      bool          `json:"should_notify"`
	NotificationLevel SeverityLevel `json:"notification_level,omitempty"`
	UserMessage       string        `json:"user_message,omitempty"`
}

// ActionType represents different types of actions that can be taken for errors.
type ActionType string

const (
	ActionRetry    ActionType = "retry"
	ActionFallback ActionType = "fallback"
	ActionFail     ActionType = "fail"
	ActionIgnore   ActionType = "ignore"
	ActionCircuit  ActionType = "circuit_break"
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
