// Package config provides custom error types for the config package.
package config

import (
	"errors"
	"fmt"
)

// Error codes for configuration operations.
const (
	ErrCodeInvalidConfig      = "invalid_config"
	ErrCodeLoadFailed         = "load_failed"
	ErrCodeValidationFailed   = "validation_failed"
	ErrCodeFileNotFound        = "file_not_found"
	ErrCodeFileReadError       = "file_read_error"
	ErrCodeUnmarshalError      = "unmarshal_error"
	ErrCodeProviderNotFound    = "provider_not_found"
	ErrCodeProviderError       = "provider_error"
	ErrCodeRequiredFieldMissing = "required_field_missing"
	ErrCodeInvalidValue        = "invalid_value"
)

// ConfigError represents an error that occurred during configuration operations.
type ConfigError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Code    string // error code for programmatic handling
	Message string // human-readable message
	Field   string // field that caused the error (if applicable)
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	if e.Message != "" {
		if e.Field != "" {
			return fmt.Sprintf("config %s [field: %s]: %s (code: %s)", e.Op, e.Field, e.Message, e.Code)
		}
		return fmt.Sprintf("config %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("config %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("config %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// NewConfigError creates a new ConfigError.
func NewConfigError(op, code string, err error) *ConfigError {
	return &ConfigError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewConfigErrorWithMessage creates a new ConfigError with a custom message.
func NewConfigErrorWithMessage(op, code, message string, err error) *ConfigError {
	return &ConfigError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewConfigErrorWithField creates a new ConfigError with a field name.
func NewConfigErrorWithField(op, code, field, message string, err error) *ConfigError {
	return &ConfigError{
		Op:      op,
		Code:    code,
		Field:   field,
		Message: message,
		Err:     err,
	}
}

// IsConfigError checks if an error is a ConfigError.
func IsConfigError(err error) bool {
	var configErr *ConfigError
	return errors.As(err, &configErr)
}

// AsConfigError attempts to convert an error to a ConfigError.
func AsConfigError(err error) (*ConfigError, bool) {
	var configErr *ConfigError
	if errors.As(err, &configErr) {
		return configErr, true
	}
	return nil, false
}
