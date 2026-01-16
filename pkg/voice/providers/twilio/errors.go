package twilio

import (
	"fmt"
)

// Error codes for Twilio voice operations.
const (
	// Twilio-specific errors.
	ErrCodeTwilioRateLimit          = "twilio_rate_limit"
	ErrCodeTwilioInvalidConfig      = "twilio_invalid_config"
	ErrCodeTwilioNetworkError       = "twilio_network_error"
	ErrCodeTwilioTimeout            = "twilio_timeout"
	ErrCodeTwilioAuthError          = "twilio_auth_error"
	ErrCodeTwilioCallFailed         = "twilio_call_failed"
	ErrCodeTwilioStreamFailed       = "twilio_stream_failed"
	ErrCodeTwilioWebhookError       = "twilio_webhook_error"
	ErrCodeTwilioTranscriptionError = "twilio_transcription_error"
	ErrCodeInvalidSignature         = "invalid_signature"
)

// TwilioError represents an error that occurred during Twilio voice operations.
// It includes an operation name, underlying error, and error code for programmatic handling.
type TwilioError struct {
	Op      string
	Err     error
	Code    string
	Message string
	Details map[string]any
}

// Error implements the error interface.
func (e *TwilioError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("voice/twilio %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("voice/twilio %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("voice/twilio %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *TwilioError) Unwrap() error {
	return e.Err
}

// NewTwilioError creates a new TwilioError.
func NewTwilioError(op, code string, err interface{}) *TwilioError {
	var errVal error
	var message string

	switch v := err.(type) {
	case error:
		errVal = v
	case string:
		message = v
		errVal = fmt.Errorf("%s", v)
	default:
		message = fmt.Sprintf("%v", v)
		errVal = fmt.Errorf("%v", v)
	}

	return &TwilioError{
		Op:      op,
		Code:    code,
		Err:     errVal,
		Message: message,
	}
}

// NewTwilioErrorWithMessage creates a new TwilioError with a custom message.
func NewTwilioErrorWithMessage(op, code, message string, err error) *TwilioError {
	return &TwilioError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewTwilioErrorWithDetails creates a new TwilioError with additional details.
func NewTwilioErrorWithDetails(op, code, message string, err error, details map[string]any) *TwilioError {
	return &TwilioError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
		Details: details,
	}
}

// MapTwilioError maps Twilio SDK errors to Beluga error types.
func MapTwilioError(op string, err error) *TwilioError {
	if err == nil {
		return nil
	}

	// Check if it's already a TwilioError
	var twilioErr *TwilioError
	if twilioErr != nil {
		return twilioErr
	}

	// Map common Twilio error patterns
	// Note: Twilio Go SDK error types would be checked here
	// For now, we'll use a generic mapping
	errStr := err.Error()

	// Map based on error message patterns
	switch {
	case contains(errStr, "rate limit") || contains(errStr, "429"):
		return NewTwilioError(op, ErrCodeTwilioRateLimit, err)
	case contains(errStr, "401") || contains(errStr, "unauthorized") || contains(errStr, "authentication"):
		return NewTwilioError(op, ErrCodeTwilioAuthError, err)
	case contains(errStr, "timeout") || contains(errStr, "deadline"):
		return NewTwilioError(op, ErrCodeTwilioTimeout, err)
	case contains(errStr, "network") || contains(errStr, "connection"):
		return NewTwilioError(op, ErrCodeTwilioNetworkError, err)
	default:
		return NewTwilioError(op, ErrCodeTwilioCallFailed, err)
	}
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
