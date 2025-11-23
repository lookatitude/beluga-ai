package transport

import (
	"errors"
	"fmt"
)

// Error codes for Transport operations
const (
	// General errors
	ErrCodeInvalidConfig = "invalid_config"
	ErrCodeInternalError = "internal_error"
	ErrCodeInvalidInput  = "invalid_input"
	ErrCodeTimeout       = "timeout"

	// Connection errors
	ErrCodeNotConnected      = "not_connected"
	ErrCodeConnectionFailed  = "connection_failed"
	ErrCodeConnectionTimeout = "connection_timeout"
	ErrCodeDisconnected      = "disconnected"

	// Provider-specific errors
	ErrCodeUnsupportedProvider = "unsupported_provider"

	// Network errors
	ErrCodeNetworkError  = "network_error"
	ErrCodeProtocolError = "protocol_error"

	// Context errors
	ErrCodeContextCanceled = "context_canceled"
	ErrCodeContextTimeout  = "context_timeout"
)

// TransportError represents an error that occurred during Transport operations.
// It includes an operation name, underlying error, and error code for programmatic handling.
type TransportError struct {
	Op      string                 // Operation that failed (e.g., "Connect", "SendAudio")
	Err     error                  // Underlying error
	Code    string                 // Error code for programmatic handling
	Message string                 // Human-readable error message
	Details map[string]interface{} // Additional error details
}

// Error implements the error interface
func (e *TransportError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("transport %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("transport %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("transport %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error
func (e *TransportError) Unwrap() error {
	return e.Err
}

// NewTransportError creates a new TransportError
func NewTransportError(op, code string, err error) *TransportError {
	return &TransportError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewTransportErrorWithMessage creates a new TransportError with a custom message
func NewTransportErrorWithMessage(op, code, message string, err error) *TransportError {
	return &TransportError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewTransportErrorWithDetails creates a new TransportError with additional details
func NewTransportErrorWithDetails(op, code, message string, err error, details map[string]interface{}) *TransportError {
	return &TransportError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
		Details: details,
	}
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var transportErr *TransportError
	if errors.As(err, &transportErr) {
		switch transportErr.Code {
		case ErrCodeTimeout, ErrCodeNetworkError, ErrCodeConnectionFailed, ErrCodeConnectionTimeout:
			return true
		default:
			return false
		}
	}

	return false
}
