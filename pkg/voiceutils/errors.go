// Package voiceutils provides standardized error handling for voice processing packages.
package voiceutils

import (
	"errors"
	"fmt"
)

// ErrorCode represents standardized error codes for voice processing.
type ErrorCode string

const (
	// ErrorCodeInvalidInput indicates invalid input parameters.
	ErrorCodeInvalidInput ErrorCode = "invalid_input"

	// ErrorCodeInvalidFormat indicates invalid audio format.
	ErrorCodeInvalidFormat ErrorCode = "invalid_format"

	// ErrorCodeUnsupportedCodec indicates an unsupported audio codec.
	ErrorCodeUnsupportedCodec ErrorCode = "unsupported_codec"

	// ErrorCodeConnectionFailed indicates a connection failure.
	ErrorCodeConnectionFailed ErrorCode = "connection_failed"

	// ErrorCodeTimeout indicates a timeout occurred.
	ErrorCodeTimeout ErrorCode = "timeout"

	// ErrorCodeStreamClosed indicates the stream was closed.
	ErrorCodeStreamClosed ErrorCode = "stream_closed"

	// ErrorCodeBufferOverflow indicates a buffer overflow.
	ErrorCodeBufferOverflow ErrorCode = "buffer_overflow"

	// ErrorCodeRateLimited indicates rate limiting was applied.
	ErrorCodeRateLimited ErrorCode = "rate_limited"

	// ErrorCodeCircuitOpen indicates the circuit breaker is open.
	ErrorCodeCircuitOpen ErrorCode = "circuit_open"

	// ErrorCodeProviderError indicates a provider-specific error.
	ErrorCodeProviderError ErrorCode = "provider_error"

	// ErrorCodeInternalError indicates an internal system error.
	ErrorCodeInternalError ErrorCode = "internal_error"
)

// Static base errors for dynamic error wrapping (err113 compliance).
var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrInvalidFormat    = errors.New("invalid audio format")
	ErrUnsupportedCodec = errors.New("unsupported codec")
	ErrConnectionFailed = errors.New("connection failed")
	ErrTimeout          = errors.New("operation timed out")
	ErrStreamClosed     = errors.New("stream closed")
	ErrBufferOverflow   = errors.New("buffer overflow")
	ErrRateLimited      = errors.New("rate limited")
	ErrCircuitOpen      = errors.New("circuit breaker open")
	ErrProviderError    = errors.New("provider error")
)

// VoiceError represents a standardized error in the voiceutils package.
// It follows the Op/Err/Code pattern used across all Beluga AI packages.
type VoiceError struct {
	Err     error
	Context map[string]any
	Op      string
	Code    ErrorCode
	Message string
}

// Error implements the error interface.
func (e *VoiceError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("voiceutils %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("voiceutils %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("voiceutils %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *VoiceError) Unwrap() error {
	return e.Err
}

// NewVoiceError creates a VoiceError following the Op/Err/Code pattern.
func NewVoiceError(op string, code ErrorCode, message string, err error) *VoiceError {
	return &VoiceError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
		Context: make(map[string]any),
	}
}

// NewInvalidInputError creates a new invalid input error.
func NewInvalidInputError(op, message string, err error) *VoiceError {
	return NewVoiceError(op, ErrorCodeInvalidInput, message, err)
}

// NewInvalidFormatError creates a new invalid format error.
func NewInvalidFormatError(op, message string, err error) *VoiceError {
	return NewVoiceError(op, ErrorCodeInvalidFormat, message, err)
}

// NewUnsupportedCodecError creates a new unsupported codec error.
func NewUnsupportedCodecError(op, codec string) *VoiceError {
	return NewVoiceError(op, ErrorCodeUnsupportedCodec, fmt.Sprintf("codec '%s' is not supported", codec), ErrUnsupportedCodec)
}

// NewConnectionError creates a new connection error.
func NewConnectionError(op, message string, err error) *VoiceError {
	return NewVoiceError(op, ErrorCodeConnectionFailed, message, err)
}

// NewTimeoutError creates a new timeout error.
func NewTimeoutError(op, message string, err error) *VoiceError {
	return NewVoiceError(op, ErrorCodeTimeout, message, err)
}

// NewStreamClosedError creates a new stream closed error.
func NewStreamClosedError(op, message string) *VoiceError {
	return NewVoiceError(op, ErrorCodeStreamClosed, message, ErrStreamClosed)
}

// NewBufferOverflowError creates a new buffer overflow error.
func NewBufferOverflowError(op string, bufferSize, dataSize int) *VoiceError {
	e := NewVoiceError(op, ErrorCodeBufferOverflow,
		fmt.Sprintf("buffer overflow: buffer size %d, data size %d", bufferSize, dataSize),
		ErrBufferOverflow)
	e.Context["buffer_size"] = bufferSize
	e.Context["data_size"] = dataSize
	return e
}

// NewRateLimitError creates a new rate limit error.
func NewRateLimitError(op, retryAfter string) *VoiceError {
	e := NewVoiceError(op, ErrorCodeRateLimited, "rate limit exceeded", ErrRateLimited)
	if retryAfter != "" {
		e.Context["retry_after"] = retryAfter
	}
	return e
}

// NewCircuitOpenError creates a new circuit open error.
func NewCircuitOpenError(op string) *VoiceError {
	return NewVoiceError(op, ErrorCodeCircuitOpen, "circuit breaker is open", ErrCircuitOpen)
}

// NewProviderError creates a new provider-specific error.
func NewProviderError(op, provider, message string, err error) *VoiceError {
	e := NewVoiceError(op, ErrorCodeProviderError, message, err)
	e.Context["provider"] = provider
	return e
}

// NewInternalError creates a new internal error.
func NewInternalError(op, message string, err error) *VoiceError {
	return NewVoiceError(op, ErrorCodeInternalError, message, err)
}

// AddContext adds context information to a VoiceError.
func (e *VoiceError) AddContext(key string, value any) *VoiceError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
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

// GetErrorCode extracts the error code from an error if it's a VoiceError.
func GetErrorCode(err error) (ErrorCode, bool) {
	if voiceErr, ok := AsVoiceError(err); ok {
		return voiceErr.Code, true
	}
	return "", false
}

// IsErrorCode checks if an error has a specific error code.
func IsErrorCode(err error, code ErrorCode) bool {
	if voiceErr, ok := AsVoiceError(err); ok {
		return voiceErr.Code == code
	}
	return false
}

// WrapError wraps an error with additional context.
func WrapError(err error, op, message string) error {
	if err == nil {
		return nil
	}
	return NewVoiceError(op, ErrorCodeInternalError, message, err)
}
