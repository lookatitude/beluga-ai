// Package voice provides custom error types for the voice package.
package voice

import (
	"errors"
	"fmt"
)

// Error codes for voice operations.
const (
	ErrCodeInvalidConfig      = "invalid_config"
	ErrCodeInvalidInput       = "invalid_input"
	ErrCodeProviderNotFound   = "provider_not_found"
	ErrCodeProviderError      = "provider_error"
	ErrCodeNetworkError       = "network_error"
	ErrCodeTimeout            = "timeout"
	ErrCodeAuthentication     = "authentication_error"
	ErrCodeInvalidAudioFormat = "invalid_audio_format"
	ErrCodeContextCanceled    = "context_canceled"
	ErrCodeContextTimeout     = "context_timeout"
	ErrCodeInternalError      = "internal_error"
)

// VoiceError represents an error that occurred during voice operations.
type VoiceError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Code    string // error code for programmatic handling
	Message string // human-readable message
}

// Error implements the error interface.
func (e *VoiceError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("voice %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("voice %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("voice %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *VoiceError) Unwrap() error {
	return e.Err
}

// NewVoiceError creates a new VoiceError.
func NewVoiceError(op, code string, err error) *VoiceError {
	return &VoiceError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewVoiceErrorWithMessage creates a new VoiceError with a custom message.
func NewVoiceErrorWithMessage(op, code, message string, err error) *VoiceError {
	return &VoiceError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsVoiceError checks if an error is a VoiceError.
func IsVoiceError(err error) bool {
	var voiceErr *VoiceError
	return errors.As(err, &voiceErr)
}

// AsVoiceError attempts to convert an error to a VoiceError.
func AsVoiceError(err error) (*VoiceError, bool) {
	var voiceErr *VoiceError
	if errors.As(err, &voiceErr) {
		return voiceErr, true
	}
	return nil, false
}
