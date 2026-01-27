package s2s

import (
	"errors"
	"fmt"
	"net/http"
)

// Error codes for S2S operations.
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

	// Audio validation errors.
	ErrCodeInvalidAudioFormat  = "invalid_audio_format"
	ErrCodeInvalidAudioQuality = "invalid_audio_quality"
)

// S2SError represents an error that occurred during S2S operations.
// It includes an operation name, underlying error, and error code for programmatic handling.
type S2SError struct {
	Err     error
	Details map[string]any
	Op      string
	Code    string
	Message string
}

// Error implements the error interface.
func (e *S2SError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("s2s %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("s2s %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("s2s %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *S2SError) Unwrap() error {
	return e.Err
}

// NewS2SError creates a new S2SError.
func NewS2SError(op, code string, err error) *S2SError {
	return &S2SError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewS2SErrorWithMessage creates a new S2SError with a custom message.
func NewS2SErrorWithMessage(op, code, message string, err error) *S2SError {
	return &S2SError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewS2SErrorWithDetails creates a new S2SError with additional details.
func NewS2SErrorWithDetails(op, code, message string, err error, details map[string]any) *S2SError {
	return &S2SError{
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

	var s2sErr *S2SError
	if errors.As(err, &s2sErr) {
		switch s2sErr.Code {
		case ErrCodeNetworkError, ErrCodeTimeout, ErrCodeRateLimit,
			ErrCodeInternalError, ErrCodeStreamError, ErrCodeStreamTimeout:
			return true
		default:
			return false
		}
	}

	return false
}

// ErrorFromHTTPStatus creates an S2SError from an HTTP status code.
func ErrorFromHTTPStatus(op string, statusCode int, err error) *S2SError {
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

	return NewS2SErrorWithMessage(op, code, message, err)
}
