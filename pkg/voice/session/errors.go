package session

import (
	"errors"
	"fmt"
)

// Error codes for Session operations.
const (
	// General errors.
	ErrCodeInvalidConfig = "invalid_config"
	ErrCodeInternalError = "internal_error"
	ErrCodeInvalidState  = "invalid_state"
	ErrCodeTimeout       = "timeout"

	// Session lifecycle errors.
	ErrCodeSessionNotFound      = "session_not_found"
	ErrCodeSessionAlreadyActive = "session_already_active"
	ErrCodeSessionNotActive     = "session_not_active"
	ErrCodeSessionExpired       = "session_expired"

	// Context errors.
	ErrCodeContextCanceled = "context_canceled"
	ErrCodeContextTimeout  = "context_timeout"

	// Agent integration error codes.
	ErrCodeAgentNotSet      = "agent_not_set"
	ErrCodeAgentInvalid     = "agent_invalid"
	ErrCodeStreamError      = "stream_error"
	ErrCodeContextError     = "context_error"
	ErrCodeInterruptionError = "interruption_error"
)

// SessionError represents an error that occurred during Session operations.
// It includes an operation name, underlying error, and error code for programmatic handling.
type SessionError struct {
	Err     error
	Details map[string]any
	Op      string
	Code    string
	Message string
}

// Error implements the error interface.
func (e *SessionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("session %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("session %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("session %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *SessionError) Unwrap() error {
	return e.Err
}

// NewSessionError creates a new SessionError.
func NewSessionError(op, code string, err error) *SessionError {
	return &SessionError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewSessionErrorWithMessage creates a new SessionError with a custom message.
func NewSessionErrorWithMessage(op, code, message string, err error) *SessionError {
	return &SessionError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewSessionErrorWithDetails creates a new SessionError with additional details.
func NewSessionErrorWithDetails(op, code, message string, err error, details map[string]any) *SessionError {
	return &SessionError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
		Details: details,
	}
}

// IsRetryableError checks if an error is retryable.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var sessionErr *SessionError
	if errors.As(err, &sessionErr) {
		switch sessionErr.Code {
		case ErrCodeTimeout, ErrCodeContextTimeout:
			return true
		default:
			return false
		}
	}

	return false
}

// NewAgentIntegrationError creates a new SessionError for agent integration operations.
// It follows the Op/Err/Code pattern for consistency.
func NewAgentIntegrationError(op, code string, err error) *SessionError {
	return NewSessionError(op, code, err)
}

// WrapAgentIntegrationError wraps an existing error as a SessionError for agent integration.
func WrapAgentIntegrationError(op, code string, err error) *SessionError {
	if err == nil {
		return nil
	}
	return NewAgentIntegrationError(op, code, err)
}
