// Package retrievers provides custom error types for the retrievers package.
package retrievers

import (
	"fmt"
	"time"
)

// Error codes for programmatic error handling
const (
	ErrCodeInvalidConfig    = "invalid_config"
	ErrCodeInvalidInput     = "invalid_input"
	ErrCodeRetrievalFailed  = "retrieval_failed"
	ErrCodeEmbeddingFailed  = "embedding_failed"
	ErrCodeVectorStoreError = "vector_store_error"
	ErrCodeTimeout          = "timeout"
	ErrCodeRateLimit        = "rate_limit"
	ErrCodeNetworkError     = "network_error"
)

// RetrieverError represents an error that occurred during retrieval operations.
type RetrieverError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Code    string // error code for programmatic handling
	Message string // human-readable message
}

func (e *RetrieverError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("retriever %s: %s", e.Op, e.Message)
	}
	return fmt.Sprintf("retriever %s: %v", e.Op, e.Err)
}

func (e *RetrieverError) Unwrap() error {
	return e.Err
}

// NewRetrieverError creates a new RetrieverError.
func NewRetrieverError(op string, err error, code string) *RetrieverError {
	return &RetrieverError{
		Op:   op,
		Err:  err,
		Code: code,
	}
}

// NewRetrieverErrorWithMessage creates a new RetrieverError with a custom message.
func NewRetrieverErrorWithMessage(op string, err error, code string, message string) *RetrieverError {
	return &RetrieverError{
		Op:      op,
		Err:     err,
		Code:    code,
		Message: message,
	}
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field string      // field that failed validation
	Value interface{} // value that failed validation
	Msg   string      // validation error message
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s' with value '%v': %s", e.Field, e.Value, e.Msg)
}

// TimeoutError represents a timeout error.
type TimeoutError struct {
	Op      string        // operation that timed out
	Timeout time.Duration // timeout duration
	Err     error         // underlying error
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("retriever %s timed out after %v: %v", e.Op, e.Timeout, e.Err)
}

func (e *TimeoutError) Unwrap() error {
	return e.Err
}

// NewTimeoutError creates a new TimeoutError.
func NewTimeoutError(op string, timeout time.Duration, err error) *TimeoutError {
	return &TimeoutError{
		Op:      op,
		Timeout: timeout,
		Err:     err,
	}
}
