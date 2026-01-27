package voiceagent

import (
	"errors"
	"fmt"
)

// Error codes for the convenience voice agent package.
const (
	ErrCodeMissingSTT      = "missing_stt"
	ErrCodeMissingTTS      = "missing_tts"
	ErrCodeSTTCreation     = "stt_creation_failed"
	ErrCodeTTSCreation     = "tts_creation_failed"
	ErrCodeVADCreation     = "vad_creation_failed"
	ErrCodeAgentCreation   = "agent_creation_failed"
	ErrCodeMemoryCreation  = "memory_creation_failed"
	ErrCodeSessionCreation = "session_creation_failed"
	ErrCodeTranscription   = "transcription_failed"
	ErrCodeSynthesis       = "synthesis_failed"
	ErrCodeInvalidConfig   = "invalid_config"
	ErrCodeExecution       = "execution_failed"
	ErrCodeShutdown        = "shutdown_failed"
)

// Error represents a convenience voice agent error following the Op/Err/Code pattern.
type Error struct {
	Err     error
	Fields  map[string]any
	Op      string
	Code    string
	Message string
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("convenience/voiceagent %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("convenience/voiceagent %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("convenience/voiceagent %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error for error wrapping.
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new Error following the Op/Err/Code pattern.
func NewError(op, code string, err error) *Error {
	return &Error{
		Op:     op,
		Code:   code,
		Err:    err,
		Fields: make(map[string]any),
	}
}

// NewErrorWithMessage creates a new Error with a custom message.
func NewErrorWithMessage(op, code, message string, err error) *Error {
	return &Error{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
		Fields:  make(map[string]any),
	}
}

// WithField adds a context field to the error.
func (e *Error) WithField(key string, value any) *Error {
	if e.Fields == nil {
		e.Fields = make(map[string]any)
	}
	e.Fields[key] = value
	return e
}

// Common error variables.
var (
	ErrMissingSTT     = errors.New("STT provider is required")
	ErrMissingTTS     = errors.New("TTS provider is required")
	ErrInvalidConfig  = errors.New("invalid configuration")
	ErrSessionStopped = errors.New("session has been stopped")
)

// IsError checks if an error is a convenience voice agent Error.
func IsError(err error) bool {
	var vaErr *Error
	return errors.As(err, &vaErr)
}

// GetErrorCode extracts the error code from an Error if present.
func GetErrorCode(err error) string {
	var vaErr *Error
	if errors.As(err, &vaErr) {
		return vaErr.Code
	}
	return ""
}
