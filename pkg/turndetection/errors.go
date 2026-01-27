package turndetection

import (
	"errors"
	"fmt"
)

// Error codes for Turn Detection operations.
const (
	// General errors.
	ErrCodeInvalidConfig = "invalid_config"
	ErrCodeInternalError = "internal_error"
	ErrCodeInvalidInput  = "invalid_input"
	ErrCodeTimeout       = "timeout"

	// Provider-specific errors.
	ErrCodeUnsupportedProvider = "unsupported_provider"
	ErrCodeModelLoadFailed     = "model_load_failed"
	ErrCodeModelNotFound       = "model_not_found"

	// Processing errors.
	ErrCodeProcessingError = "processing_error"
)

// TurnDetectionError represents an error that occurred during Turn Detection operations.
// It includes an operation name, underlying error, and error code for programmatic handling.
type TurnDetectionError struct {
	Err     error
	Details map[string]any
	Op      string
	Code    string
	Message string
}

// Error implements the error interface.
func (e *TurnDetectionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("turndetection %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("turndetection %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("turndetection %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *TurnDetectionError) Unwrap() error {
	return e.Err
}

// NewTurnDetectionError creates a new TurnDetectionError.
func NewTurnDetectionError(op, code string, err error) *TurnDetectionError {
	return &TurnDetectionError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewTurnDetectionErrorWithMessage creates a new TurnDetectionError with a custom message.
func NewTurnDetectionErrorWithMessage(op, code, message string, err error) *TurnDetectionError {
	return &TurnDetectionError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewTurnDetectionErrorWithDetails creates a new TurnDetectionError with additional details.
func NewTurnDetectionErrorWithDetails(op, code, message string, err error, details map[string]any) *TurnDetectionError {
	return &TurnDetectionError{
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

	var turnErr *TurnDetectionError
	if errors.As(err, &turnErr) {
		switch turnErr.Code {
		case ErrCodeTimeout, ErrCodeProcessingError:
			return true
		default:
			return false
		}
	}

	return false
}
