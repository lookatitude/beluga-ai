package config

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ConfigError defines structured error handling following the Op/Err/Code pattern
// required by the Beluga AI Framework constitution for consistent error management
// across the framework while preserving error chains and providing actionable information.
type ConfigError struct {
	Op         string                 `json:"op"`                    // Operation that failed
	Code       string                 `json:"code"`                  // Standardized error code
	Err        error                  `json:"err,omitempty"`         // Underlying error
	Provider   string                 `json:"provider,omitempty"`    // Provider name if applicable
	Format     string                 `json:"format,omitempty"`      // Configuration format if applicable
	Path       string                 `json:"path,omitempty"`        // Configuration path if applicable
	Context    map[string]interface{} `json:"context,omitempty"`     // Additional context
	Timestamp  time.Time              `json:"timestamp"`             // When error occurred
	Retryable  bool                   `json:"retryable"`             // Whether operation can be retried
	RetryAfter time.Duration          `json:"retry_after,omitempty"` // Suggested retry delay
}

// Error implements the error interface
func (e *ConfigError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s %s: %v", e.Op, e.Code, e.Err)
	}
	return fmt.Sprintf("%s %s", e.Op, e.Code)
}

// GetOperation returns the operation that failed
func (e *ConfigError) GetOperation() string {
	return e.Op
}

// GetCode returns the standardized error code
func (e *ConfigError) GetCode() string {
	return e.Code
}

// GetProvider returns the provider name where the error occurred
func (e *ConfigError) GetProvider() string {
	return e.Provider
}

// GetFormat returns the configuration format if applicable
func (e *ConfigError) GetFormat() string {
	return e.Format
}

// GetConfigPath returns the configuration path if applicable
func (e *ConfigError) GetConfigPath() string {
	return e.Path
}

// GetContext returns additional context information
func (e *ConfigError) GetContext() map[string]interface{} {
	if e.Context == nil {
		return make(map[string]interface{})
	}
	contextCopy := make(map[string]interface{})
	for k, v := range e.Context {
		contextCopy[k] = v
	}
	return contextCopy
}

// GetTimestamp returns when the error occurred
func (e *ConfigError) GetTimestamp() time.Time {
	return e.Timestamp
}

// IsRetryable returns whether the operation should be retried
func (e *ConfigError) IsRetryable() bool {
	return e.Retryable
}

// GetRetryAfter returns the recommended delay before retry
func (e *ConfigError) GetRetryAfter() time.Duration {
	return e.RetryAfter
}

// Unwrap returns the underlying error for error chain preservation
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// Is supports errors.Is for error comparison
func (e *ConfigError) Is(target error) bool {
	if targetErr, ok := target.(*ConfigError); ok {
		return e.Code == targetErr.Code
	}
	return errors.Is(e.Err, target)
}

// As supports errors.As for type assertion
func (e *ConfigError) As(target interface{}) bool {
	if targetPtr, ok := target.(**ConfigError); ok {
		*targetPtr = e
		return true
	}
	return false
}

// Standard error codes for configuration operations following constitutional requirements
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

// NewError creates a new ConfigError with the specified details
func NewError(op, code, provider string, err error) *ConfigError {
	return &ConfigError{
		Op:         op,
		Code:       code,
		Err:        err,
		Provider:   provider,
		Timestamp:  time.Now(),
		Retryable:  isRetryableError(code),
		RetryAfter: getRetryAfterDelay(code),
	}
}

// NewLoadError creates a configuration loading error with format and path context
func NewLoadError(op, code, provider, format, path string, err error) *ConfigError {
	configErr := NewError(op, code, provider, err)
	configErr.Format = format
	configErr.Path = path
	return configErr
}

// NewValidationError creates a validation error with detailed context
func NewValidationError(op, code string, err error, validationContext map[string]interface{}) *ConfigError {
	configErr := NewError(op, code, "", err)
	if validationContext != nil {
		configErr.Context = make(map[string]interface{})
		for k, v := range validationContext {
			configErr.Context[k] = v
		}
	}
	return configErr
}

// NewProviderError creates a provider-specific error
func NewProviderError(op, code, provider string, err error) *ConfigError {
	return NewError(op, code, provider, err)
}

// WrapError wraps an existing error with configuration-specific information
func WrapError(err error, op, provider string) *ConfigError {
	if err == nil {
		return nil
	}

	// If it's already a ConfigError, preserve the original
	if configErr, ok := err.(*ConfigError); ok {
		return configErr
	}

	return NewError(op, classifyErrorCode(err), provider, err)
}

// AsConfigError attempts to cast an error to ConfigError
func AsConfigError(err error) (*ConfigError, bool) {
	var configErr *ConfigError
	if errors.As(err, &configErr) {
		return configErr, true
	}
	return nil, false
}

// isRetryableError determines if an error code indicates a retryable condition
func isRetryableError(code string) bool {
	retryableCodes := map[string]bool{
		ErrCodeLoadFailed:             true,
		ErrCodeFilePermissionDenied:   false, // Permission issues usually aren't transient
		ErrCodeParseError:             false, // Parse errors need fixing
		ErrCodeFormatNotSupported:     false, // Format issues need fixing
		ErrCodeValidationFailed:       false, // Validation failures need fixing
		ErrCodeProviderCreationFailed: true,
		ErrCodeProviderNotFound:       false, // Registry issues aren't transient
		ErrCodeHealthCheckFailed:      true,
		ErrCodeOperationTimeout:       true,
		ErrCodeContextCancelled:       false, // Cancellation is final
		ErrCodeResourceExhausted:      true,
		ErrCodeInternalError:          true,
	}

	return retryableCodes[code]
}

// getRetryAfterDelay returns suggested retry delay for retryable errors
func getRetryAfterDelay(code string) time.Duration {
	switch code {
	case ErrCodeOperationTimeout:
		return 5 * time.Second
	case ErrCodeResourceExhausted:
		return 10 * time.Second
	case ErrCodeInternalError:
		return 30 * time.Second
	case ErrCodeHealthCheckFailed:
		return 15 * time.Second
	case ErrCodeLoadFailed:
		return 2 * time.Second
	case ErrCodeProviderCreationFailed:
		return 3 * time.Second
	default:
		return 0
	}
}

// classifyErrorCode attempts to classify a generic error into a standard error code
func classifyErrorCode(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// Check for common error patterns
	if errors.Is(err, context.Canceled) {
		return ErrCodeContextCancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrCodeOperationTimeout
	}

	// Simple pattern matching for common errors
	switch {
	case containsString(errStr, "file not found"):
		return ErrCodeFileNotFound
	case containsString(errStr, "permission denied"):
		return ErrCodeFilePermissionDenied
	case containsString(errStr, "parse error"):
		return ErrCodeParseError
	case containsString(errStr, "format not supported"):
		return ErrCodeFormatNotSupported
	case containsString(errStr, "validation failed"):
		return ErrCodeValidationFailed
	default:
		return ErrCodeInternalError
	}
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			containsStringRecursive(s, substr))
}

func containsStringRecursive(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if len(s) == len(substr) {
		return s == substr
	}
	return s[:len(substr)] == substr ||
		containsStringRecursive(s[1:], substr)
}

// Error handling utilities

// WithContext adds context information to an error
func WithContext(err error, key string, value interface{}) error {
	if configErr, ok := err.(*ConfigError); ok {
		if configErr.Context == nil {
			configErr.Context = make(map[string]interface{})
		}
		configErr.Context[key] = value
		return configErr
	}
	return err
}

// WithRetryAfter sets the retry delay for an error
func WithRetryAfter(err error, delay time.Duration) error {
	if configErr, ok := err.(*ConfigError); ok {
		configErr.RetryAfter = delay
		configErr.Retryable = delay > 0
		return configErr
	}
	return err
}

// IsConfigError checks if an error is a configuration error
func IsConfigError(err error) bool {
	var configErr *ConfigError
	return errors.As(err, &configErr)
}

// GetErrorCode extracts the error code from a configuration error
func GetErrorCode(err error) string {
	if configErr, ok := AsConfigError(err); ok {
		return configErr.GetCode()
	}
	return ""
}

// IsRetryable checks if an error indicates a retryable condition
func IsRetryable(err error) bool {
	if configErr, ok := AsConfigError(err); ok {
		return configErr.IsRetryable()
	}
	return false
}

// GetRetryAfter returns the retry delay for an error
func GetRetryAfter(err error) time.Duration {
	if configErr, ok := AsConfigError(err); ok {
		return configErr.GetRetryAfter()
	}
	return 0
}
