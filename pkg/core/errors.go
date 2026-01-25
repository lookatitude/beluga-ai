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
// It follows the Op/Err/Code pattern used across all Beluga AI packages.
type FrameworkError struct {
	Err     error
	Context map[string]any
	Op      string
	Code    string
	Message string
}

// Error implements the error interface.
func (e *FrameworkError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("core %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("core %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("core %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *FrameworkError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new validation error.
// Use this for errors related to invalid input, configuration, or data validation.
//
// Parameters:
//   - op: Operation that failed
//   - message: Human-readable error message
//   - err: Underlying error that caused the validation failure (can be nil)
//
// Returns:
//   - *FrameworkError: A new validation error instance
//
// Example:
//
//	err := core.NewValidationError("validate_config", "invalid API key format", nil)
//
// Example usage can be found in examples/core/basic/main.go.
func NewValidationError(op, message string, err error) *FrameworkError {
	return &FrameworkError{
		Op:      op,
		Code:    string(ErrorCodeInvalidInput),
		Message: message,
		Err:     err,
	}
}

// NewNetworkError creates a new network error.
// Use this for errors related to network operations (connection failures, timeouts, etc.).
//
// Parameters:
//   - op: Operation that failed
//   - message: Human-readable error message
//   - err: Underlying network error (can be nil)
//
// Returns:
//   - *FrameworkError: A new network error instance
//
// Example:
//
//	err := core.NewNetworkError("connect", "connection timeout", timeoutErr)
//
// Example usage can be found in examples/core/basic/main.go.
func NewNetworkError(op, message string, err error) *FrameworkError {
	return &FrameworkError{
		Op:      op,
		Code:    string(ErrorCodeTimeout),
		Message: message,
		Err:     err,
	}
}

// NewAuthenticationError creates a new authentication error.
// Use this for errors related to authentication or authorization failures.
//
// Parameters:
//   - op: Operation that failed
//   - message: Human-readable error message
//   - err: Underlying authentication error (can be nil)
//
// Returns:
//   - *FrameworkError: A new authentication error instance
//
// Example:
//
//	err := core.NewAuthenticationError("authenticate", "invalid API key", authErr)
//
// Example usage can be found in examples/core/basic/main.go.
func NewAuthenticationError(op, message string, err error) *FrameworkError {
	return &FrameworkError{
		Op:      op,
		Code:    string(ErrorCodeUnauthorized),
		Message: message,
		Err:     err,
	}
}

// NewInternalError creates a new internal error.
// Use this for unexpected internal system errors that should be logged and investigated.
//
// Parameters:
//   - op: Operation that failed
//   - message: Human-readable error message
//   - err: Underlying internal error (can be nil)
//
// Returns:
//   - *FrameworkError: A new internal error instance
//
// Example:
//
//	err := core.NewInternalError("process", "unexpected state", stateErr)
//
// Example usage can be found in examples/core/basic/main.go.
func NewInternalError(op, message string, err error) *FrameworkError {
	return &FrameworkError{
		Op:      op,
		Code:    string(ErrorCodeInternalError),
		Message: message,
		Err:     err,
	}
}

// NewConfigurationError creates a new configuration error.
// Use this for errors related to configuration loading, validation, or parsing.
//
// Parameters:
//   - op: Operation that failed
//   - message: Human-readable error message
//   - err: Underlying configuration error (can be nil)
//
// Returns:
//   - *FrameworkError: A new configuration error instance
//
// Example:
//
//	err := core.NewConfigurationError("load_config", "missing required config key", nil)
//
// Example usage can be found in examples/core/basic/main.go.
func NewConfigurationError(op, message string, err error) *FrameworkError {
	return &FrameworkError{
		Op:      op,
		Code:    string(ErrorCodeInvalidInput),
		Message: message,
		Err:     err,
	}
}

// IsErrorType checks if an error is of a specific FrameworkError type.
// Deprecated: Use error codes instead. This function is kept for backward compatibility.
func IsErrorType(err error, errorType ErrorType) bool {
	var fwErr *FrameworkError
	if AsFrameworkError(err, &fwErr) {
		// Map ErrorType to error codes for backward compatibility
		switch errorType {
		case ErrorTypeValidation:
			return fwErr.Code == string(ErrorCodeInvalidInput)
		case ErrorTypeNetwork:
			return fwErr.Code == string(ErrorCodeTimeout)
		case ErrorTypeAuthentication:
			return fwErr.Code == string(ErrorCodeUnauthorized)
		case ErrorTypeInternal:
			return fwErr.Code == string(ErrorCodeInternalError)
		case ErrorTypeConfiguration:
			return fwErr.Code == string(ErrorCodeInvalidInput)
		}
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
func WrapError(err error, op, message string) error {
	if err == nil {
		return nil
	}
	return &FrameworkError{
		Op:      op,
		Code:    string(ErrorCodeInternalError),
		Message: message,
		Err:     err,
	}
}

// ErrorCode represents standardized error codes for programmatic error handling.
type ErrorCode string

const (
	// ErrorCodeInvalidInput indicates invalid input parameters.
	ErrorCodeInvalidInput ErrorCode = "invalid_input"

	// ErrorCodeNotFound indicates a resource was not found.
	ErrorCodeNotFound ErrorCode = "not_found"

	// ErrorCodeUnauthorized indicates unauthorized access.
	ErrorCodeUnauthorized ErrorCode = "unauthorized"

	// ErrorCodeTimeout indicates a timeout occurred.
	ErrorCodeTimeout ErrorCode = "timeout"

	// ErrorCodeRateLimited indicates rate limiting was applied.
	ErrorCodeRateLimited ErrorCode = "rate_limited"

	// ErrorCodeInternalError indicates an internal system error.
	ErrorCodeInternalError ErrorCode = "internal_error"
)

// NewFrameworkError creates a FrameworkError following the Op/Err/Code pattern.
func NewFrameworkError(op string, code ErrorCode, message string, err error) *FrameworkError {
	return &FrameworkError{
		Op:      op,
		Code:    string(code),
		Message: message,
		Err:     err,
		Context: make(map[string]any),
	}
}

// NewFrameworkErrorWithCode creates a FrameworkError with a specific error code.
// Deprecated: Use NewFrameworkError instead.
func NewFrameworkErrorWithCode(errorType ErrorType, code ErrorCode, message string, cause error) *FrameworkError {
	return NewFrameworkError("operation", code, message, cause)
}

// AddContext adds context information to a FrameworkError.
func (e *FrameworkError) AddContext(key string, value any) *FrameworkError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}
