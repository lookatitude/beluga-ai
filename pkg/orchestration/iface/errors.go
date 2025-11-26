package iface

import (
	"errors"
	"fmt"
)

// OrchestratorError represents errors that occur during orchestration operations.
type OrchestratorError struct {
	Op   string // operation that failed
	Err  error  // underlying error
	Code string // error code for programmatic handling
}

func (e *OrchestratorError) Error() string {
	return fmt.Sprintf("orchestrator %s: %v", e.Op, e.Err)
}

func (e *OrchestratorError) Unwrap() error {
	return e.Err
}

// Error codes for orchestration operations.
const (
	ErrCodeInvalidConfig      = "invalid_config"
	ErrCodeExecutionFailed    = "execution_failed"
	ErrCodeTimeout            = "timeout"
	ErrCodeDependencyFailed   = "dependency_failed"
	ErrCodeResourceExhausted  = "resource_exhausted"
	ErrCodeInvalidState       = "invalid_state"
	ErrCodeNotFound           = "not_found"
	ErrCodeCircuitBreakerOpen = "circuit_breaker_open"
	ErrCodeRateLimitExceeded  = "rate_limit_exceeded"
	ErrCodeInvalidInput       = "invalid_input"
	ErrCodeWorkflowDeadlock   = "workflow_deadlock"
	ErrCodeTaskCancelled      = "task_canceled"
	ErrCodeMaxRetriesExceeded = "max_retries_exceeded"
)

// NewOrchestratorError creates a new orchestrator error.
func NewOrchestratorError(op string, err error, code string) *OrchestratorError {
	return &OrchestratorError{
		Op:   op,
		Err:  err,
		Code: code,
	}
}

// Common error constructors.
func ErrInvalidConfig(op string, err error) *OrchestratorError {
	return NewOrchestratorError(op, err, ErrCodeInvalidConfig)
}

func ErrExecutionFailed(op string, err error) *OrchestratorError {
	return NewOrchestratorError(op, err, ErrCodeExecutionFailed)
}

func ErrTimeout(op string, err error) *OrchestratorError {
	return NewOrchestratorError(op, err, ErrCodeTimeout)
}

func ErrDependencyFailed(op, dependency string, err error) *OrchestratorError {
	return NewOrchestratorError(op, fmt.Errorf("dependency %s failed: %w", dependency, err), ErrCodeDependencyFailed)
}

func ErrResourceExhausted(op, resource string) *OrchestratorError {
	return NewOrchestratorError(op, fmt.Errorf("resource %s exhausted", resource), ErrCodeResourceExhausted)
}

func ErrInvalidState(op, currentState, expectedState string) *OrchestratorError {
	return NewOrchestratorError(op, fmt.Errorf("invalid state: got %s, expected %s", currentState, expectedState), ErrCodeInvalidState)
}

func ErrNotFound(op, resource string) *OrchestratorError {
	return NewOrchestratorError(op, fmt.Errorf("resource %s not found", resource), ErrCodeNotFound)
}

func ErrCircuitBreakerOpen(op string) *OrchestratorError {
	return NewOrchestratorError(op, errors.New("circuit breaker is open"), ErrCodeCircuitBreakerOpen)
}

func ErrRateLimitExceeded(op string, limit int) *OrchestratorError {
	return NewOrchestratorError(op, fmt.Errorf("rate limit exceeded: %d", limit), ErrCodeRateLimitExceeded)
}

func ErrInvalidInput(op, inputType, reason string) *OrchestratorError {
	return NewOrchestratorError(op, fmt.Errorf("invalid input type %s: %s", inputType, reason), ErrCodeInvalidInput)
}

func ErrWorkflowDeadlock(op, workflowID string) *OrchestratorError {
	return NewOrchestratorError(op, fmt.Errorf("workflow deadlock detected in %s", workflowID), ErrCodeWorkflowDeadlock)
}

func ErrTaskCancelled(op, taskID string) *OrchestratorError {
	return NewOrchestratorError(op, fmt.Errorf("task %s was canceled", taskID), ErrCodeTaskCancelled)
}

func ErrMaxRetriesExceeded(op, taskID string, maxRetries int) *OrchestratorError {
	return NewOrchestratorError(op, fmt.Errorf("task %s exceeded maximum retries (%d)", taskID, maxRetries), ErrCodeMaxRetriesExceeded)
}

// IsRetryable checks if an error is retryable based on its error code.
func IsRetryable(err error) bool {
	var orchErr *OrchestratorError
	if errors.As(err, &orchErr) {
		switch orchErr.Code {
		case ErrCodeTimeout, ErrCodeDependencyFailed, ErrCodeResourceExhausted, ErrCodeCircuitBreakerOpen, ErrCodeRateLimitExceeded:
			return true
		case ErrCodeInvalidConfig, ErrCodeInvalidState, ErrCodeNotFound, ErrCodeInvalidInput, ErrCodeWorkflowDeadlock:
			return false
		default:
			return false
		}
	}
	return false
}
