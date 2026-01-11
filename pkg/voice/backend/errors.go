package backend

import (
	"errors"
	"fmt"
)

// Error codes for voice backend operations.
const (
	// General errors.
	ErrCodeInvalidConfig        = "invalid_config"
	ErrCodeProviderNotFound     = "provider_not_found"
	ErrCodeConnectionFailed     = "connection_failed"
	ErrCodeConnectionTimeout    = "connection_timeout"
	ErrCodeSessionNotFound      = "session_not_found"
	ErrCodeSessionLimitExceeded = "session_limit_exceeded"
	ErrCodeRateLimitExceeded    = "rate_limit_exceeded"
	ErrCodeAuthenticationFailed = "authentication_failed"
	ErrCodeAuthorizationFailed   = "authorization_failed"
	ErrCodePipelineError        = "pipeline_error"
	ErrCodeAgentError           = "agent_error"
	ErrCodeTimeout              = "timeout"
	ErrCodeContextCanceled      = "context_canceled"
	ErrCodeInvalidFormat        = "invalid_format"
	ErrCodeConversionFailed     = "conversion_failed"
)

// BackendError represents an error that occurred during voice backend operations.
// It includes an operation name, underlying error, and error code for programmatic handling.
type BackendError struct {
	Op      string
	Err     error
	Code    string
	Message string
	Details map[string]any
}

// Error implements the error interface.
func (e *BackendError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("voice/backend %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("voice/backend %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("voice/backend %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *BackendError) Unwrap() error {
	return e.Err
}

// NewBackendError creates a new BackendError.
func NewBackendError(op, code string, err error) *BackendError {
	return &BackendError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewBackendErrorWithMessage creates a new BackendError with a custom message.
func NewBackendErrorWithMessage(op, code, message string, err error) *BackendError {
	return &BackendError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewBackendErrorWithDetails creates a new BackendError with additional details.
func NewBackendErrorWithDetails(op, code, message string, err error, details map[string]any) *BackendError {
	return &BackendError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
		Details: details,
	}
}

// WrapError wraps an error with additional context.
func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}

	var backendErr *BackendError
	if errors.As(err, &backendErr) {
		// Already a BackendError, just update the operation
		backendErr.Op = op
		return backendErr
	}

	// Create new BackendError with default code
	return NewBackendError(op, ErrCodePipelineError, err)
}

// IsError checks if an error is a BackendError.
func IsError(err error) bool {
	var backendErr *BackendError
	return errors.As(err, &backendErr)
}

// AsError extracts a BackendError from an error if it exists.
func AsError(err error) *BackendError {
	var backendErr *BackendError
	if errors.As(err, &backendErr) {
		return backendErr
	}
	return nil
}

// IsRetryableError checks if an error is transient and can be retried.
func IsRetryableError(err error) bool {
	if backendErr := AsError(err); backendErr != nil {
		switch backendErr.Code {
		case ErrCodeConnectionTimeout, ErrCodeRateLimitExceeded, ErrCodeConnectionFailed:
			return true
		default:
			return false
		}
	}
	// Default to not retry for unknown errors
	return false
}
