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
// It unwraps the error chain to find any ConfigError with the matching code.
func IsConfigError(err error, code string) bool {
	for err != nil {
		if cfgErr, ok := err.(*ConfigError); ok {
			if cfgErr.Code == code {
				return true
			}
			// Continue unwrapping to check inner ConfigErrors
			err = cfgErr.Unwrap()
		} else if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}

// AsConfigError attempts to cast an error to ConfigError.
// It returns the first (outermost) ConfigError found in the error chain.
// It unwraps ConfigErrors to find nested ConfigErrors, but does not unwrap
// regular errors (like fmt.Errorf) that might wrap ConfigErrors.
func AsConfigError(err error, target **ConfigError) bool {
	for err != nil {
		if cfgErr, ok := err.(*ConfigError); ok {
			// Set target to the first ConfigError found (outermost)
			if *target == nil {
				*target = cfgErr
			}
			// Unwrap to check for nested ConfigErrors
			err = cfgErr.Unwrap()
			if err == nil {
				return true
			}
			// Continue loop to check nested ConfigErrors
			continue
		}
		// For non-ConfigError types, stop here - don't unwrap
		// This prevents finding ConfigErrors wrapped in regular errors (like fmt.Errorf)
		break
	}
	return *target != nil
}
