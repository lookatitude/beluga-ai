// Package embeddings provides custom error types for the embeddings package.
package embeddings

import (
	"errors"
	"fmt"
)

// Error codes for embedding operations.
const (
	ErrCodeInvalidConfig     = "invalid_config"
	ErrCodeInvalidInput      = "invalid_input"
	ErrCodeEmbeddingFailed   = "embedding_failed"
	ErrCodeProviderNotFound  = "provider_not_found"
	ErrCodeProviderError     = "provider_error"
	ErrCodeProviderDisabled  = "provider_disabled"
	ErrCodeNetworkError      = "network_error"
	ErrCodeConnectionFailed  = "connection_failed"
	ErrCodeTimeout           = "timeout"
	ErrCodeRateLimit         = "rate_limit"
	ErrCodeAuthentication    = "authentication_error"
	ErrCodeInvalidDimension  = "invalid_dimension"
	ErrCodeInvalidParameters = "invalid_parameters"
	ErrCodeBatchSizeExceeded = "batch_size_exceeded"
	ErrCodeContextCanceled   = "context_canceled"
	ErrCodeContextTimeout    = "context_timeout"
)

// Static base errors for dynamic error wrapping (err113 compliance).
var (
	ErrInvalidConfig     = errors.New("invalid configuration")
	ErrInvalidInput      = errors.New("invalid input")
	ErrEmbeddingFailed   = errors.New("embedding operation failed")
	ErrProviderNotFound  = errors.New("provider not found")
	ErrProviderError     = errors.New("provider error")
	ErrProviderDisabled  = errors.New("provider disabled")
	ErrNetworkError      = errors.New("network error")
	ErrConnectionFailed  = errors.New("connection failed")
	ErrTimeout           = errors.New("operation timed out")
	ErrRateLimit         = errors.New("rate limit exceeded")
	ErrAuthentication    = errors.New("authentication failed")
	ErrInvalidDimension  = errors.New("invalid dimension")
	ErrInvalidParameters = errors.New("invalid parameters")
	ErrBatchSizeExceeded = errors.New("batch size exceeded")
	ErrContextCanceled   = errors.New("context canceled")
	ErrContextTimeout    = errors.New("context timeout")
)

// EmbeddingError represents an error that occurred during embedding operations.
type EmbeddingError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Code    string // error code for programmatic handling
	Message string // human-readable message
}

// Error implements the error interface.
func (e *EmbeddingError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("embeddings %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("embeddings %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("embeddings %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *EmbeddingError) Unwrap() error {
	return e.Err
}

// NewEmbeddingError creates a new EmbeddingError.
func NewEmbeddingError(op, code string, err error) *EmbeddingError {
	return &EmbeddingError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewEmbeddingErrorWithMessage creates a new EmbeddingError with a custom message.
func NewEmbeddingErrorWithMessage(op, code, message string, err error) *EmbeddingError {
	return &EmbeddingError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsEmbeddingError checks if an error is an EmbeddingError.
func IsEmbeddingError(err error) bool {
	var embeddingErr *EmbeddingError
	return errors.As(err, &embeddingErr)
}

// AsEmbeddingError attempts to convert an error to an EmbeddingError.
func AsEmbeddingError(err error) (*EmbeddingError, bool) {
	var embeddingErr *EmbeddingError
	if errors.As(err, &embeddingErr) {
		return embeddingErr, true
	}
	return nil, false
}

// GetErrorCode extracts the error code from an error if it's an EmbeddingError.
func GetErrorCode(err error) (string, bool) {
	if embErr, ok := AsEmbeddingError(err); ok {
		return embErr.Code, true
	}
	return "", false
}

// IsErrorCode checks if an error has a specific error code.
func IsErrorCode(err error, code string) bool {
	if embErr, ok := AsEmbeddingError(err); ok {
		return embErr.Code == code
	}
	return false
}

// NewInvalidConfigError creates a new invalid config error.
func NewInvalidConfigError(op, message string, err error) *EmbeddingError {
	return NewEmbeddingErrorWithMessage(op, ErrCodeInvalidConfig, message, err)
}

// NewProviderNotFoundError creates a new provider not found error.
func NewProviderNotFoundError(op, providerName string) *EmbeddingError {
	return NewEmbeddingErrorWithMessage(op, ErrCodeProviderNotFound, fmt.Sprintf("provider '%s' not found", providerName), ErrProviderNotFound)
}

// NewProviderDisabledError creates a new provider disabled error.
func NewProviderDisabledError(op, providerName string) *EmbeddingError {
	return NewEmbeddingErrorWithMessage(op, ErrCodeProviderDisabled, fmt.Sprintf("provider '%s' is disabled", providerName), ErrProviderDisabled)
}

// NewConnectionFailedError creates a new connection failed error.
func NewConnectionFailedError(op, message string, err error) *EmbeddingError {
	return NewEmbeddingErrorWithMessage(op, ErrCodeConnectionFailed, message, err)
}

// NewInvalidParametersError creates a new invalid parameters error.
func NewInvalidParametersError(op, message string, err error) *EmbeddingError {
	return NewEmbeddingErrorWithMessage(op, ErrCodeInvalidParameters, message, err)
}

// WrapError wraps an error with additional context.
func WrapError(err error, op, code, message string) *EmbeddingError {
	if err == nil {
		return nil
	}
	return NewEmbeddingErrorWithMessage(op, code, message, err)
}
