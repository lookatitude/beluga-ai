package noise

import (
	"errors"
	"fmt"
)

// Error codes for Noise Cancellation operations.
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
	ErrCodeFrameSizeError  = "frame_size_error"
	ErrCodeSampleRateError = "sample_rate_error"

	// Context errors.
	ErrCodeContextCanceled = "context_canceled"
	ErrCodeContextTimeout  = "context_timeout"
)

// NoiseCancellationError represents an error that occurred during Noise Cancellation operations.
// It includes an operation name, underlying error, and error code for programmatic handling.
type NoiseCancellationError struct {
	Err     error
	Details map[string]any
	Op      string
	Code    string
	Message string
}

// Error implements the error interface.
func (e *NoiseCancellationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("noise %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("noise %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("noise %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *NoiseCancellationError) Unwrap() error {
	return e.Err
}

// NewNoiseCancellationError creates a new NoiseCancellationError.
func NewNoiseCancellationError(op, code string, err error) *NoiseCancellationError {
	return &NoiseCancellationError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewNoiseCancellationErrorWithMessage creates a new NoiseCancellationError with a custom message.
func NewNoiseCancellationErrorWithMessage(op, code, message string, err error) *NoiseCancellationError {
	return &NoiseCancellationError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewNoiseCancellationErrorWithDetails creates a new NoiseCancellationError with additional details.
func NewNoiseCancellationErrorWithDetails(op, code, message string, err error, details map[string]any) *NoiseCancellationError {
	return &NoiseCancellationError{
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

	var noiseErr *NoiseCancellationError
	if errors.As(err, &noiseErr) {
		switch noiseErr.Code {
		case ErrCodeTimeout, ErrCodeProcessingError:
			return true
		default:
			return false
		}
	}

	return false
}
