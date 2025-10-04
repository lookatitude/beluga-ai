package llms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Error codes for LLM operations
const (
	// General errors
	ErrCodeInvalidConfig  = "invalid_config"
	ErrCodeNetworkError   = "network_error"
	ErrCodeTimeout        = "timeout"
	ErrCodeRateLimit      = "rate_limit"
	ErrCodeQuotaExceeded  = "quota_exceeded"
	ErrCodeAuthentication = "authentication_error"
	ErrCodeAuthorization  = "authorization_error"
	ErrCodeNotFound       = "not_found"
	ErrCodeInternalError  = "internal_error"
	ErrCodeInvalidInput   = "invalid_input"

	// Provider-specific errors
	ErrCodeUnsupportedProvider = "unsupported_provider"
	ErrCodeInvalidModel        = "invalid_model"
	ErrCodeModelNotAvailable   = "model_not_available"

	// Request/Response errors
	ErrCodeInvalidRequest    = "invalid_request"
	ErrCodeInvalidResponse   = "invalid_response"
	ErrCodeEmptyResponse     = "empty_response"
	ErrCodeMalformedResponse = "malformed_response"

	// Streaming errors
	ErrCodeStreamError   = "stream_error"
	ErrCodeStreamTimeout = "stream_timeout"
	ErrCodeStreamClosed  = "stream_closed"

	// Tool calling errors
	ErrCodeToolCallError      = "tool_call_error"
	ErrCodeToolNotFound       = "tool_not_found"
	ErrCodeToolExecutionError = "tool_execution_error"

	// Context errors
	ErrCodeContextCanceled = "context_canceled"
	ErrCodeContextTimeout  = "context_timeout"
)

// LLMError represents an error that occurred during LLM operations.
// It includes an operation name, underlying error, and error code for programmatic handling.
type LLMError struct {
	Op      string                 // Operation that failed (e.g., "generate", "stream")
	Err     error                  // Underlying error
	Code    string                 // Error code for programmatic handling
	Message string                 // Human-readable error message
	Details map[string]interface{} // Additional error details
}

// Error implements the error interface
func (e *LLMError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("llms %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("llms %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("llms %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error
func (e *LLMError) Unwrap() error {
	return e.Err
}

// NewLLMError creates a new LLMError
func NewLLMError(op, code string, err error) *LLMError {
	return &LLMError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewLLMErrorWithMessage creates a new LLMError with a custom message
func NewLLMErrorWithMessage(op, code, message string, err error) *LLMError {
	return &LLMError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewLLMErrorWithDetails creates a new LLMError with additional details
func NewLLMErrorWithDetails(op, code, message string, err error, details map[string]interface{}) *LLMError {
	return &LLMError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
		Details: details,
	}
}

// IsLLMError checks if an error is an LLMError
func IsLLMError(err error) bool {
	var llmErr *LLMError
	return errors.As(err, &llmErr)
}

// GetLLMError extracts an LLMError from an error if it exists
func GetLLMError(err error) *LLMError {
	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return llmErr
	}
	return nil
}

// GetLLMErrorCode extracts the error code from an LLMError
func GetLLMErrorCode(err error) string {
	llmErr := GetLLMError(err)
	if llmErr != nil {
		return llmErr.Code
	}
	return ""
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a context error
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false // Context errors are not retryable
	}

	// Check LLM error codes
	code := GetLLMErrorCode(err)
	switch code {
	case ErrCodeRateLimit, ErrCodeNetworkError, ErrCodeTimeout, ErrCodeInternalError:
		return true
	case ErrCodeQuotaExceeded, ErrCodeAuthentication, ErrCodeAuthorization, ErrCodeInvalidConfig, ErrCodeInvalidRequest:
		return false
	default:
		// For unknown errors, check if they're HTTP errors that might be retryable
		var httpErr interface{ StatusCode() int }
		if errors.As(err, &httpErr) {
			statusCode := httpErr.StatusCode()
			return statusCode >= 500 || statusCode == http.StatusTooManyRequests
		}
		// Default to retryable for unknown errors
		return true
	}
}

// WrapError wraps an error with additional context
func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}

	if llmErr, ok := err.(*LLMError); ok {
		// Already an LLMError, just update the operation
		llmErr.Op = op
		return llmErr
	}

	// Try to map common errors to LLM error codes
	var code string
	switch {
	case errors.Is(err, context.Canceled):
		code = ErrCodeContextCanceled
	case errors.Is(err, context.DeadlineExceeded):
		code = ErrCodeContextTimeout
	default:
		code = ErrCodeInternalError
	}

	return NewLLMError(op, code, err)
}

// MapHTTPError maps HTTP status codes to LLM error codes
func MapHTTPError(op string, statusCode int, err error) *LLMError {
	var code string
	var message string

	switch statusCode {
	case http.StatusUnauthorized:
		code = ErrCodeAuthentication
		message = "authentication failed"
	case http.StatusForbidden:
		code = ErrCodeAuthorization
		message = "authorization failed"
	case http.StatusNotFound:
		code = ErrCodeNotFound
		message = "resource not found"
	case http.StatusTooManyRequests:
		code = ErrCodeRateLimit
		message = "rate limit exceeded"
	case http.StatusBadRequest:
		code = ErrCodeInvalidRequest
		message = "invalid request"
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		code = ErrCodeInternalError
		message = "internal server error"
	default:
		code = ErrCodeNetworkError
		message = "network error"
	}

	return NewLLMErrorWithMessage(op, code, message, err)
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// ConfigValidationError represents multiple validation errors
type ConfigValidationError struct {
	Errors []ValidationError
}

// Error implements the error interface
func (e *ConfigValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "configuration validation failed"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("configuration validation failed with %d errors", len(e.Errors))
}

// AddError adds a validation error
func (e *ConfigValidationError) AddError(field string, value interface{}, message string) {
	e.Errors = append(e.Errors, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

// HasErrors checks if there are any validation errors
func (e *ConfigValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// ProviderError represents provider-specific errors
type ProviderError struct {
	Provider string
	Op       string
	Code     string
	Message  string
	Err      error
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider %s %s: %s (code: %s)", e.Provider, e.Op, e.Message, e.Code)
}

// Unwrap returns the underlying error
func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new ProviderError
func NewProviderError(provider, op, code, message string, err error) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Op:       op,
		Code:     code,
		Message:  message,
		Err:      err,
	}
}

// StreamError represents streaming-specific errors
type StreamError struct {
	Op      string
	Code    string
	Message string
	Err     error
}

// Error implements the error interface
func (e *StreamError) Error() string {
	return fmt.Sprintf("stream %s: %s (code: %s)", e.Op, e.Message, e.Code)
}

// Unwrap returns the underlying error
func (e *StreamError) Unwrap() error {
	return e.Err
}

// NewStreamError creates a new StreamError
func NewStreamError(op, code, message string, err error) *StreamError {
	return &StreamError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}
