// Package orchestration provides custom error types for the orchestration package.
package orchestration

import (
	"errors"
	"fmt"
)

// Error codes for orchestration operations.
const (
	ErrCodeInvalidConfig     = "invalid_config"
	ErrCodeInvalidInput      = "invalid_input"
	ErrCodeSchedulingFailed  = "scheduling_failed"
	ErrCodeExecutionFailed   = "execution_failed"
	ErrCodeWorkflowNotFound  = "workflow_not_found"
	ErrCodeWorkflowError     = "workflow_error"
	ErrCodeTaskNotFound      = "task_not_found"
	ErrCodeTaskError         = "task_error"
	ErrCodeDependencyError   = "dependency_error"
	ErrCodeTimeout           = "timeout"
	ErrCodeContextCanceled   = "context_canceled"
	ErrCodeContextTimeout    = "context_timeout"
	ErrCodeResourceExhausted = "resource_exhausted"
	ErrCodeInvalidState      = "invalid_state"
	ErrCodeStateTransition   = "state_transition_error"
)

// OrchestrationError represents an error that occurred during orchestration operations.
type OrchestrationError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Code    string // error code for programmatic handling
	Message string // human-readable message
}

// Error implements the error interface.
func (e *OrchestrationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("orchestration %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("orchestration %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("orchestration %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *OrchestrationError) Unwrap() error {
	return e.Err
}

// NewOrchestrationError creates a new OrchestrationError.
func NewOrchestrationError(op, code string, err error) *OrchestrationError {
	return &OrchestrationError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewOrchestrationErrorWithMessage creates a new OrchestrationError with a custom message.
func NewOrchestrationErrorWithMessage(op, code, message string, err error) *OrchestrationError {
	return &OrchestrationError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsOrchestrationError checks if an error is an OrchestrationError.
func IsOrchestrationError(err error) bool {
	var orchErr *OrchestrationError
	return errors.As(err, &orchErr)
}

// AsOrchestrationError attempts to convert an error to an OrchestrationError.
func AsOrchestrationError(err error) (*OrchestrationError, bool) {
	var orchErr *OrchestrationError
	if errors.As(err, &orchErr) {
		return orchErr, true
	}
	return nil, false
}
