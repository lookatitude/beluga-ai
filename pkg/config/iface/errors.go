package iface

import "fmt"

// ConfigError represents errors specific to configuration operations.
// It provides structured error information for programmatic error handling.
type ConfigError struct {
	Code    string // Error code for programmatic handling
	Message string // Human-readable error message
	Cause   error  // Underlying error that caused this error
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error for error wrapping compatibility.
func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// NewConfigError creates a new ConfigError with the given code and message.
func NewConfigError(code, message string, args ...interface{}) *ConfigError {
	return &ConfigError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
	}
}

// WrapError wraps an existing error with config context.
func WrapError(cause error, code, message string, args ...interface{}) *ConfigError {
	return &ConfigError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		Cause:   cause,
	}
}

// Common error codes
const (
	ErrCodeInvalidConfig       = "invalid_config"
	ErrCodeValidationFailed    = "validation_failed"
	ErrCodeFileNotFound        = "file_not_found"
	ErrCodeParseFailed         = "parse_failed"
	ErrCodeUnsupportedFormat   = "unsupported_format"
	ErrCodeMissingRequired     = "missing_required"
	ErrCodeInvalidProvider     = "invalid_provider"
	ErrCodeInvalidParameters   = "invalid_parameters"
	ErrCodeLoadFailed          = "load_failed"
	ErrCodeSaveFailed          = "save_failed"
	ErrCodeProviderUnavailable = "provider_unavailable"
	ErrCodeRemoteLoadTimeout   = "remote_load_timeout"
	ErrCodeAllProvidersFailed  = "all_providers_failed"
	ErrCodeConfigNotFound      = "config_not_found"
	ErrCodeKeyNotFound         = "key_not_found"
	ErrCodeInvalidFormat       = "invalid_format"
)

// IsConfigError checks if an error is a ConfigError with the given code.
func IsConfigError(err error, code string) bool {
	var cfgErr *ConfigError
	if !AsConfigError(err, &cfgErr) {
		return false
	}
	return cfgErr.Code == code
}

// AsConfigError attempts to cast an error to ConfigError.
func AsConfigError(err error, target **ConfigError) bool {
	for err != nil {
		if cfgErr, ok := err.(*ConfigError); ok {
			*target = cfgErr
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}
