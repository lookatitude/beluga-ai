// Package core provides standardized error handling for the Beluga AI framework.
package core

import (
	"fmt"
)

// ErrorType represents different categories of errors in the framework.
type ErrorType string

const (
	// ErrorTypeValidation indicates validation-related errors (invalid input, configuration, etc.)
	ErrorTypeValidation ErrorType = "validation"

	// ErrorTypeNetwork indicates network-related errors (connection failures, timeouts, etc.)
	ErrorTypeNetwork ErrorType = "network"

	// ErrorTypeAuthentication indicates authentication/authorization errors
	ErrorTypeAuthentication ErrorType = "authentication"

	// ErrorTypeRateLimit indicates rate limiting errors
	ErrorTypeRateLimit ErrorType = "rate_limit"

	// ErrorTypeInternal indicates internal system errors
	ErrorTypeInternal ErrorType = "internal"

	// ErrorTypeExternal indicates errors from external services/APIs
	ErrorTypeExternal ErrorType = "external"

	// ErrorTypeConfiguration indicates configuration-related errors
	ErrorTypeConfiguration ErrorType = "configuration"
)

// FrameworkError represents a standardized error in the Beluga AI framework.
// T017: Enhanced to ensure full Op/Err/Code pattern compliance
type FrameworkError struct {
	Op      string                 `json:"op"`                // Operation that failed (constitutional requirement)
	Err     error                  `json:"err"`               // Underlying error (constitutional requirement)
	Code    string                 `json:"code"`              // Error code for programmatic handling (constitutional requirement)
	Type    ErrorType              `json:"type"`              // Error category
	Message string                 `json:"message"`           // Human-readable message
	Context map[string]interface{} `json:"context,omitempty"` // Additional context information
}

// Error implements the error interface with Op/Err/Code pattern.
func (e *FrameworkError) Error() string {
	if e.Op != "" && e.Err != nil {
		return fmt.Sprintf("core.%s: %s (code: %s): %v", e.Op, e.Message, e.Code, e.Err)
	} else if e.Err != nil {
		return fmt.Sprintf("[%s] %s (code: %s): %v", e.Type, e.Message, e.Code, e.Err)
	} else if e.Op != "" {
		return fmt.Sprintf("core.%s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	return fmt.Sprintf("[%s] %s (code: %s)", e.Type, e.Message, e.Code)
}

// Unwrap returns the underlying cause of the error.
func (e *FrameworkError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new validation error.
func NewValidationError(message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    ErrorTypeValidation,
		Message: message,
		Err:     cause,
	}
}

// NewNetworkError creates a new network error.
func NewNetworkError(message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    ErrorTypeNetwork,
		Message: message,
		Err:     cause,
	}
}

// NewAuthenticationError creates a new authentication error.
func NewAuthenticationError(message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    ErrorTypeAuthentication,
		Message: message,
		Err:     cause,
	}
}

// NewInternalError creates a new internal error.
func NewInternalError(message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    ErrorTypeInternal,
		Message: message,
		Err:     cause,
	}
}

// NewConfigurationError creates a new configuration error.
func NewConfigurationError(message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    ErrorTypeConfiguration,
		Message: message,
		Err:     cause,
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
	if fwErr, ok := err.(*FrameworkError); ok {
		*target = fwErr
		return true
	}

	// Check if it's wrapped
	if cause := UnwrapError(err); cause != nil {
		if fwErr, ok := cause.(*FrameworkError); ok {
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
		Err:     err,
	}
}

// ErrorCode represents standardized error codes for programmatic error handling.
type ErrorCode string

const (
	// ErrorCodeInvalidInput indicates invalid input parameters
	ErrorCodeInvalidInput ErrorCode = "INVALID_INPUT"

	// ErrorCodeNotFound indicates a resource was not found
	ErrorCodeNotFound ErrorCode = "NOT_FOUND"

	// ErrorCodeUnauthorized indicates unauthorized access
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"

	// ErrorCodeTimeout indicates a timeout occurred
	ErrorCodeTimeout ErrorCode = "TIMEOUT"

	// ErrorCodeRateLimited indicates rate limiting was applied
	ErrorCodeRateLimited ErrorCode = "RATE_LIMITED"

	// ErrorCodeInternalError indicates an internal system error
	ErrorCodeInternalError ErrorCode = "INTERNAL_ERROR"
)

// NewFrameworkErrorWithCode creates a FrameworkError with a specific error code.
func NewFrameworkErrorWithCode(errorType ErrorType, code ErrorCode, message string, cause error) *FrameworkError {
	return &FrameworkError{
		Type:    errorType,
		Message: message,
		Err:     cause,
		Code:    string(code),
		Context: make(map[string]interface{}),
	}
}

// AddContext adds context information to a FrameworkError.
func (e *FrameworkError) AddContext(key string, value interface{}) *FrameworkError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}
