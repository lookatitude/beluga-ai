package agent

import (
	"errors"
	"fmt"
)

// Error codes for the convenience agent package.
const (
	ErrCodeMissingLLM     = "missing_llm"
	ErrCodeLLMCreation    = "llm_creation_failed"
	ErrCodeMemoryCreation = "memory_creation_failed"
	ErrCodeAgentCreation  = "agent_creation_failed"
	ErrCodeInvalidLLMType = "invalid_llm_type"
	ErrCodeExecution      = "execution_failed"
	ErrCodeStreaming      = "streaming_failed"
	ErrCodeShutdown       = "shutdown_failed"
	ErrCodeInvalidConfig  = "invalid_config"
)

// Error represents a convenience agent error following the Op/Err/Code pattern.
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
		return fmt.Sprintf("convenience/agent %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("convenience/agent %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("convenience/agent %s: unknown error (code: %s)", e.Op, e.Code)
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
	ErrMissingLLM      = errors.New("LLM is required")
	ErrInvalidConfig   = errors.New("invalid configuration")
	ErrExecutionFailed = errors.New("execution failed")
	ErrShutdownFailed  = errors.New("shutdown failed")
)

// IsError checks if an error is a convenience agent Error.
func IsError(err error) bool {
	var agentErr *Error
	return errors.As(err, &agentErr)
}

// GetErrorCode extracts the error code from an Error if present.
func GetErrorCode(err error) string {
	var agentErr *Error
	if errors.As(err, &agentErr) {
		return agentErr.Code
	}
	return ""
}
