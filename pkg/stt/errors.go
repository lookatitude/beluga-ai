package stt

import (
	"errors"
	"fmt"
	"net/http"
)

// Error codes for STT operations.
const (
	// General errors.
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

	// Provider-specific errors.
	ErrCodeUnsupportedProvider = "unsupported_provider"
	ErrCodeInvalidModel        = "invalid_model"
	ErrCodeModelNotAvailable   = "model_not_available"

	// Request/Response errors.
	ErrCodeInvalidRequest    = "invalid_request"
	ErrCodeInvalidResponse   = "invalid_response"
	ErrCodeEmptyResponse     = "empty_response"
	ErrCodeMalformedResponse = "malformed_response"

	// Streaming errors.
	ErrCodeStreamError   = "stream_error"
	ErrCodeStreamTimeout = "stream_timeout"
	ErrCodeStreamClosed  = "stream_closed"

	// Context errors.
	ErrCodeContextCanceled = "context_canceled"
	ErrCodeContextTimeout  = "context_timeout"
)

// STTError represents an error that occurred during STT operations.
// It includes an operation name, underlying error, and error code for programmatic handling.
type STTError struct {
	Err     error
	Details map[string]any
	Op      string
	Code    string
	Message string
}

// Error implements the error interface.
func (e *STTError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("stt %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("stt %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("stt %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *STTError) Unwrap() error {
	return e.Err
}

// NewSTTError creates a new STTError.
func NewSTTError(op, code string, err error) *STTError {
	return &STTError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewSTTErrorWithMessage creates a new STTError with a custom message.
func NewSTTErrorWithMessage(op, code, message string, err error) *STTError {
	return &STTError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewSTTErrorWithDetails creates a new STTError with additional details.
func NewSTTErrorWithDetails(op, code, message string, err error, details map[string]any) *STTError {
	return &STTError{
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

	var sttErr *STTError
	if errors.As(err, &sttErr) {
		switch sttErr.Code {
		case ErrCodeNetworkError, ErrCodeTimeout, ErrCodeRateLimit,
			ErrCodeInternalError, ErrCodeStreamError, ErrCodeStreamTimeout:
			return true
		default:
			return false
		}
	}

	return false
}

// ErrorFromHTTPStatus creates an STTError from an HTTP status code.
func ErrorFromHTTPStatus(op string, statusCode int, err error) *STTError {
	var code string
	var message string

	switch statusCode {
	case http.StatusBadRequest:
		code = ErrCodeInvalidRequest
		message = "invalid request"
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
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		code = ErrCodeInternalError
		message = "server error"
	default:
		code = ErrCodeNetworkError
		message = fmt.Sprintf("HTTP error: %d", statusCode)
	}

	return NewSTTErrorWithMessage(op, code, message, err)
}
