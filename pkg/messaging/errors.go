package messaging

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Error codes for messaging operations.
const (
	// General errors.
	ErrCodeInvalidConfig  = "invalid_config"
	ErrCodeNetworkError   = "network_error"
	ErrCodeTimeout        = "timeout"
	ErrCodeRateLimit      = "rate_limit"
	ErrCodeAuthentication = "authentication_error"
	ErrCodeAuthorization  = "authorization_error"
	ErrCodeNotFound       = "not_found"
	ErrCodeInternalError  = "internal_error"
	ErrCodeInvalidInput   = "invalid_input"

	// Provider-specific errors.
	ErrCodeUnsupportedProvider = "unsupported_provider"
	ErrCodeProviderNotFound    = "provider_not_found"

	// Conversation errors.
	ErrCodeConversationNotFound = "conversation_not_found"
	ErrCodeConversationFailed   = "conversation_failed"
	ErrCodeInvalidConversation  = "invalid_conversation"
	ErrCodeCloseFailed          = "close_failed"

	// Message errors.
	ErrCodeInvalidMessage = "invalid_message"
	ErrCodeMessageFailed  = "message_failed"
	ErrCodeChannelFailed  = "channel_failed"

	// Participant errors.
	ErrCodeParticipantNotFound = "participant_not_found"
	ErrCodeInvalidParticipant  = "invalid_participant"

	// Webhook errors.
	ErrCodeInvalidWebhook   = "invalid_webhook"
	ErrCodeInvalidSignature = "invalid_signature"

	// Context errors.
	ErrCodeContextCanceled = "context_canceled"
	ErrCodeContextTimeout  = "context_timeout"

	// Shutdown errors.
	ErrCodeShutdownError = "shutdown_error"

	// Quota errors.
	ErrCodeQuotaExceeded = "quota_exceeded"
)

// MessagingError represents an error that occurred during messaging operations.
// It includes an operation name, underlying error, and error code for programmatic handling.
type MessagingError struct {
	Err     error
	Details map[string]any
	Op      string
	Code    string
	Message string
}

// Error implements the error interface.
func (e *MessagingError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("messaging %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("messaging %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("messaging %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *MessagingError) Unwrap() error {
	return e.Err
}

// NewMessagingError creates a new MessagingError.
func NewMessagingError(op, code string, err error) *MessagingError {
	return &MessagingError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewMessagingErrorWithMessage creates a new MessagingError with a custom message.
func NewMessagingErrorWithMessage(op, code, message string, err error) *MessagingError {
	return &MessagingError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewMessagingErrorWithDetails creates a new MessagingError with additional details.
func NewMessagingErrorWithDetails(op, code, message string, err error, details map[string]any) *MessagingError {
	return &MessagingError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
		Details: details,
	}
}

// IsMessagingError checks if an error is a MessagingError.
func IsMessagingError(err error) bool {
	var msgErr *MessagingError
	return errors.As(err, &msgErr)
}

// GetMessagingError extracts a MessagingError from an error if it exists.
func GetMessagingError(err error) *MessagingError {
	var msgErr *MessagingError
	if errors.As(err, &msgErr) {
		return msgErr
	}
	return nil
}

// GetMessagingErrorCode extracts the error code from a MessagingError.
func GetMessagingErrorCode(err error) string {
	msgErr := GetMessagingError(err)
	if msgErr != nil {
		return msgErr.Code
	}
	return ""
}

// IsRetryableError checks if an error is retryable.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a context error
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false // Context errors are not retryable
	}

	// Check messaging error codes
	code := GetMessagingErrorCode(err)
	switch code {
	case ErrCodeRateLimit, ErrCodeNetworkError, ErrCodeTimeout, ErrCodeInternalError:
		return true
	case ErrCodeQuotaExceeded, ErrCodeAuthentication, ErrCodeAuthorization, ErrCodeInvalidConfig, ErrCodeInvalidInput:
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

// WrapError wraps an error with additional context.
func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}

	msgErr := &MessagingError{}
	if errors.As(err, &msgErr) {
		// Already a MessagingError, just update the operation
		msgErr.Op = op
		return msgErr
	}

	// Try to map common errors to messaging error codes
	var code string
	switch {
	case errors.Is(err, context.Canceled):
		code = ErrCodeContextCanceled
	case errors.Is(err, context.DeadlineExceeded):
		code = ErrCodeContextTimeout
	default:
		code = ErrCodeInternalError
	}

	return NewMessagingError(op, code, err)
}

// MapHTTPError maps HTTP status codes to messaging error codes.
func MapHTTPError(op string, statusCode int, err error) *MessagingError {
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
		code = ErrCodeInvalidInput
		message = "invalid request"
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		code = ErrCodeInternalError
		message = "internal server error"
	default:
		code = ErrCodeNetworkError
		message = "network error"
	}

	return NewMessagingErrorWithMessage(op, code, message, err)
}
