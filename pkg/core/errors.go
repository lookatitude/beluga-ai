// Package core provides standardized error handling for the Beluga AI framework.
package core

import (
	"errors"
	"fmt"
)

// ErrorType represents different categories of errors in the framework.
type ErrorType string

const (
	// ErrorTypeValidation indicates validation-related errors (invalid input, configuration, etc.)
	ErrorTypeValidation ErrorType = "validation"

	// ErrorTypeNetwork indicates network-related errors (connection failures, timeouts, etc.)
	ErrorTypeNetwork ErrorType = "network"

	// ErrorTypeAuthentication indicates authentication/authorization errors.
	ErrorTypeAuthentication ErrorType = "authentication"

	// ErrorTypeRateLimit indicates rate limiting errors.
	ErrorTypeRateLimit ErrorType = "rate_limit"

	// ErrorTypeInternal indicates internal system errors.
	ErrorTypeInternal ErrorType = "internal"

	// ErrorTypeExternal indicates errors from external services/APIs.
	ErrorTypeExternal ErrorType = "external"

	// ErrorTypeConfiguration indicates configuration-related errors.
	ErrorTypeConfiguration ErrorType = "configuration"
)

// FrameworkError represents a standardized error in the Beluga AI framework.
type FrameworkError struct {
	Cause   error
	Context map[string]any
	Type    ErrorType
	Message string
	Code    string
}

// Error implements the error interface.
func (e *FrameworkError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *FrameworkError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new validation error.
func NewValidationError(message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    ErrorTypeValidation,
		Message: message,
		Cause:   cause,
	}
}

// NewNetworkError creates a new network error.
func NewNetworkError(message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    ErrorTypeNetwork,
		Message: message,
		Cause:   cause,
	}
}

// NewAuthenticationError creates a new authentication error.
func NewAuthenticationError(message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    ErrorTypeAuthentication,
		Message: message,
		Cause:   cause,
	}
}

// NewInternalError creates a new internal error.
func NewInternalError(message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    ErrorTypeInternal,
		Message: message,
		Cause:   cause,
	}
}

// NewConfigurationError creates a new configuration error.
func NewConfigurationError(message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    ErrorTypeConfiguration,
		Message: message,
		Cause:   cause,
	}
}

// IsErrorType checks if an error is of a specific FrameworkError type.
func IsErrorType(err error, errorType ErrorType) bool {
	var fwErr *FrameworkError
	if AsFrameworkError(err, &fwErr) {
		return fwErr.Type == errorType
	}
	return false
}

// AsFrameworkError attempts to convert an error to a FrameworkError.
func AsFrameworkError(err error, target **FrameworkError) bool {
	fwErr := &FrameworkError{}
	if errors.As(err, &fwErr) {
		*target = fwErr
		return true
	}

	// Check if it's wrapped
	if cause := UnwrapError(err); cause != nil {
		fwErr := &FrameworkError{}
		if errors.As(cause, &fwErr) {
			*target = fwErr
			return true
		}
	}

	return false
}

// UnwrapError unwraps an error to get its underlying cause.
// This is a compatibility function for Go versions before 1.13.
func UnwrapError(err error) error {
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		return unwrapper.Unwrap()
	}
	return nil
}

// WrapError wraps an error with additional context.
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return &FrameworkError{
		Type:    ErrorTypeInternal,
		Message: message,
		Cause:   err,
	}
}

// ErrorCode represents standardized error codes for programmatic error handling.
type ErrorCode string

const (
	// ErrorCodeInvalidInput indicates invalid input parameters.
	ErrorCodeInvalidInput ErrorCode = "INVALID_INPUT"

	// ErrorCodeNotFound indicates a resource was not found.
	ErrorCodeNotFound ErrorCode = "NOT_FOUND"

	// ErrorCodeUnauthorized indicates unauthorized access.
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"

	// ErrorCodeTimeout indicates a timeout occurred.
	ErrorCodeTimeout ErrorCode = "TIMEOUT"

	// ErrorCodeRateLimited indicates rate limiting was applied.
	ErrorCodeRateLimited ErrorCode = "RATE_LIMITED"

	// ErrorCodeInternalError indicates an internal system error.
	ErrorCodeInternalError ErrorCode = "INTERNAL_ERROR"
)

// NewFrameworkErrorWithCode creates a FrameworkError with a specific error code.
func NewFrameworkErrorWithCode(errorType ErrorType, code ErrorCode, message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
		Code:    string(code),
		Context: make(map[string]any),
	}
}

// AddContext adds context information to a FrameworkError.
func (e *FrameworkError) AddContext(key string, value any) *FrameworkError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}
