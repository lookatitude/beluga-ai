package vad

import (
	"errors"
	"fmt"
)

// Error codes for VAD operations
const (
	// General errors
	ErrCodeInvalidConfig = "invalid_config"
	ErrCodeInternalError = "internal_error"
	ErrCodeInvalidInput  = "invalid_input"
	ErrCodeTimeout       = "timeout"

	// Provider-specific errors
	ErrCodeUnsupportedProvider = "unsupported_provider"
	ErrCodeModelLoadFailed     = "model_load_failed"
	ErrCodeModelNotFound       = "model_not_found"

	// Processing errors
	ErrCodeProcessingError = "processing_error"
	ErrCodeFrameSizeError  = "frame_size_error"
	ErrCodeSampleRateError = "sample_rate_error"

	// Context errors
	ErrCodeContextCanceled = "context_canceled"
	ErrCodeContextTimeout  = "context_timeout"
)

// VADError represents an error that occurred during VAD operations.
// It includes an operation name, underlying error, and error code for programmatic handling.
type VADError struct {
	Op      string                 // Operation that failed (e.g., "Process", "LoadModel")
	Err     error                  // Underlying error
	Code    string                 // Error code for programmatic handling
	Message string                 // Human-readable error message
	Details map[string]interface{} // Additional error details
}

// Error implements the error interface
func (e *VADError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("vad %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("vad %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("vad %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error
func (e *VADError) Unwrap() error {
	return e.Err
}

// NewVADError creates a new VADError
func NewVADError(op, code string, err error) *VADError {
	return &VADError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewVADErrorWithMessage creates a new VADError with a custom message
func NewVADErrorWithMessage(op, code, message string, err error) *VADError {
	return &VADError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewVADErrorWithDetails creates a new VADError with additional details
func NewVADErrorWithDetails(op, code, message string, err error, details map[string]interface{}) *VADError {
	return &VADError{
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

	var vadErr *VADError
	if errors.As(err, &vadErr) {
		switch vadErr.Code {
		case ErrCodeTimeout, ErrCodeProcessingError:
			return true
		default:
			return false
		}
	}

	return false
}
