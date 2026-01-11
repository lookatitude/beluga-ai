// Package multimodal provides custom error types for multimodal operations.
package multimodal

import (
	"context"
	"errors"
	"fmt"
)

// Error codes for multimodal operations.
const (
	ErrCodeProviderNotFound     = "provider_not_found"
	ErrCodeInvalidConfig        = "invalid_config"
	ErrCodeInvalidInput         = "invalid_input"
	ErrCodeInvalidFormat        = "invalid_format"
	ErrCodeProviderError        = "provider_error"
	ErrCodeUnsupportedModality  = "unsupported_modality"
	ErrCodeTimeout              = "timeout"
	ErrCodeCancelled            = "cancelled"
	ErrCodeFileNotFound         = "file_not_found"
	ErrCodeRateLimit            = "rate_limit"
	ErrCodeQuotaExceeded        = "quota_exceeded"
	ErrCodeAuthenticationFailed = "authentication_failed"
	ErrCodeNetworkError         = "network_error"
	ErrCodeSerializationError   = "serialization_error"
	ErrCodeDeserializationError = "deserialization_error"
	ErrCodeContentTooLarge      = "content_too_large"
	ErrCodeInvalidMIMEType      = "invalid_mime_type"
	ErrCodeStreamingError       = "streaming_error"
	ErrCodeHealthCheckFailed    = "health_check_failed"
	ErrCodeRoutingError         = "routing_error"
	ErrCodeNormalizationError   = "normalization_error"
)

// MultimodalError represents an error that occurred during multimodal operations.
type MultimodalError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Code    string // error code for programmatic handling
	Message string // human-readable message
}

// Error implements the error interface.
func (e *MultimodalError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("multimodal %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("multimodal %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("multimodal %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *MultimodalError) Unwrap() error {
	return e.Err
}

// NewMultimodalError creates a new MultimodalError with the given operation, error code, and underlying error.
// The error code should be one of the ErrCode* constants defined in this package.
func NewMultimodalError(op, code string, err error) *MultimodalError {
	return &MultimodalError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewMultimodalErrorWithMessage creates a new MultimodalError with a custom human-readable message.
// This is useful when you want to provide more context than what the underlying error provides.
func NewMultimodalErrorWithMessage(op, code, message string, err error) *MultimodalError {
	return &MultimodalError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WrapError wraps a standard error as a MultimodalError.
// If err is nil, returns nil. Otherwise, creates a new MultimodalError with the given operation and error code.
func WrapError(err error, op, code string) *MultimodalError {
	if err == nil {
		return nil
	}
	return &MultimodalError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// IsMultimodalError checks if an error is a MultimodalError.
func IsMultimodalError(err error) bool {
	var mmErr *MultimodalError
	return errors.As(err, &mmErr)
}

// AsMultimodalError attempts to convert an error to a MultimodalError.
func AsMultimodalError(err error) (*MultimodalError, bool) {
	var mmErr *MultimodalError
	if errors.As(err, &mmErr) {
		return mmErr, true
	}
	return nil, false
}

// IsRetryableError checks if an error is retryable.
// Retryable errors include rate limits, network errors, and timeouts.
// Non-retryable errors include authentication failures, invalid input, and quota exceeded.
//
// Example:
//
//	if err != nil && multimodal.IsRetryableError(err) {
//	    // Retry the operation
//	    time.Sleep(1 * time.Second)
//	    return model.Process(ctx, input)
//	}
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	mmErr, ok := AsMultimodalError(err)
	if !ok {
		// Check for context errors which are generally not retryable
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		// Network errors and timeouts are generally retryable
		return true
	}

	switch mmErr.Code {
	case ErrCodeRateLimit, ErrCodeTimeout, ErrCodeNetworkError:
		return true
	case ErrCodeProviderError:
		// Provider errors might be retryable depending on the underlying error
		return true
	case ErrCodeQuotaExceeded, ErrCodeAuthenticationFailed, ErrCodeInvalidConfig,
		ErrCodeInvalidInput, ErrCodeInvalidFormat, ErrCodeUnsupportedModality,
		ErrCodeFileNotFound, ErrCodeContentTooLarge, ErrCodeInvalidMIMEType:
		return false
	default:
		return false
	}
}

// GetErrorCode extracts the error code from an error.
// Returns the error code if err is a MultimodalError, otherwise returns ErrCodeProviderError.
// This is useful for programmatic error handling based on error codes.
func GetErrorCode(err error) string {
	if err == nil {
		return ""
	}

	mmErr, ok := AsMultimodalError(err)
	if !ok {
		// Return a generic error code for non-multimodal errors
		return ErrCodeProviderError
	}

	return mmErr.Code
}

// IsErrorCode checks if an error has a specific error code.
// Returns true if the error's code matches the given code, false otherwise.
//
// Example:
//
//	if multimodal.IsErrorCode(err, multimodal.ErrCodeRateLimit) {
//	    // Handle rate limit
//	}
func IsErrorCode(err error, code string) bool {
	return GetErrorCode(err) == code
}

// WrapContextError wraps a context error (Canceled or DeadlineExceeded) as a MultimodalError.
// This is useful for consistent error handling when context cancellation occurs.
func WrapContextError(err error, op string) *MultimodalError {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return NewMultimodalError(op, ErrCodeCancelled, err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return NewMultimodalError(op, ErrCodeTimeout, err)
	}
	return WrapError(err, op, ErrCodeProviderError)
}

// WrapNetworkError wraps a network error as a MultimodalError.
// Detects network errors including DNS failures, connection refused, and timeouts.
func WrapNetworkError(err error, op string) *MultimodalError {
	if err == nil {
		return nil
	}
	// Check if it's already a MultimodalError
	if mmErr, ok := AsMultimodalError(err); ok {
		return mmErr
	}
	// Check for context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return WrapContextError(err, op)
	}
	// Check for network errors (net.Error interface)
	// Note: We can't import net here to avoid import cycles, so we check error strings
	errStr := err.Error()
	if containsAny(errStr, []string{"connection refused", "connection reset", "no such host", "network is unreachable", "timeout", "i/o timeout"}) {
		return NewMultimodalError(op, ErrCodeNetworkError, err)
	}
	// Default to provider error
	return WrapError(err, op, ErrCodeProviderError)
}

// WrapIOError wraps an I/O error as a MultimodalError.
// Handles EOF, permission denied, and other I/O errors.
func WrapIOError(err error, op string) *MultimodalError {
	if err == nil {
		return nil
	}
	// Check if it's already a MultimodalError
	if mmErr, ok := AsMultimodalError(err); ok {
		return mmErr
	}
	// Check for context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return WrapContextError(err, op)
	}
	// Check for EOF (not an error in streaming contexts, but can be in other contexts)
	if errors.Is(err, errors.New("EOF")) {
		// EOF is typically not an error in streaming, but we wrap it for consistency
		return NewMultimodalError(op, ErrCodeStreamingError, err)
	}
	// Check for file-related errors
	errStr := err.Error()
	if containsAny(errStr, []string{"no such file", "file not found", "cannot find"}) {
		return NewMultimodalError(op, ErrCodeFileNotFound, err)
	}
	if containsAny(errStr, []string{"permission denied", "access denied"}) {
		return NewMultimodalErrorWithMessage(op, ErrCodeProviderError, "permission denied", err)
	}
	// Default to provider error
	return WrapError(err, op, ErrCodeProviderError)
}

// WrapValidationError wraps a validation error as a MultimodalError.
// Used for input validation, config validation, and format validation errors.
func WrapValidationError(err error, op string, validationType string) *MultimodalError {
	if err == nil {
		return nil
	}
	// Check if it's already a MultimodalError
	if mmErr, ok := AsMultimodalError(err); ok {
		return mmErr
	}
	// Determine error code based on validation type
	var code string
	switch validationType {
	case "config":
		code = ErrCodeInvalidConfig
	case "input":
		code = ErrCodeInvalidInput
	case "format":
		code = ErrCodeInvalidFormat
	case "mime":
		code = ErrCodeInvalidMIMEType
	default:
		code = ErrCodeInvalidInput
	}
	return NewMultimodalError(op, code, err)
}

// IsContextError checks if an error is a context cancellation or deadline exceeded error.
// Returns true if the error is context.Canceled or context.DeadlineExceeded.
func IsContextError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// IsNetworkError checks if an error is a network-related error.
// Returns true if the error code is ErrCodeNetworkError or if the error string
// contains network-related keywords.
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if IsErrorCode(err, ErrCodeNetworkError) {
		return true
	}
	errStr := err.Error()
	return containsAny(errStr, []string{"connection", "network", "dns", "timeout", "refused", "reset"})
}

// IsFileError checks if an error is a file-related error.
// Returns true if the error code is ErrCodeFileNotFound or if the error string
// contains file-related keywords.
func IsFileError(err error) bool {
	if err == nil {
		return false
	}
	if IsErrorCode(err, ErrCodeFileNotFound) {
		return true
	}
	errStr := err.Error()
	return containsAny(errStr, []string{"file not found", "no such file", "cannot find", "permission denied", "access denied"})
}

// containsAny checks if a string contains any of the given substrings.
// This is a helper function for error classification.
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
