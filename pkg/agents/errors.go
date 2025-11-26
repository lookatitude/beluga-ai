package agents

import (
	"errors"
	"fmt"
)

// AgentError represents a custom error type for agent-related operations.
// It includes context about the operation that failed and wraps the underlying error.
type AgentError struct {
	Err    error
	Fields map[string]any
	Op     string
	Agent  string
	Code   string
}

// Error implements the error interface.
func (e *AgentError) Error() string {
	if e.Agent != "" {
		return fmt.Sprintf("agent %s %s: %v", e.Agent, e.Op, e.Err)
	}
	return fmt.Sprintf("agent %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error for error wrapping.
func (e *AgentError) Unwrap() error {
	return e.Err
}

// Error codes for different types of agent errors.
const (
	ErrCodeConfigInvalid     = "config_invalid"
	ErrCodeInitialization    = "initialization_failed"
	ErrCodeExecution         = "execution_failed"
	ErrCodePlanning          = "planning_failed"
	ErrCodeToolNotFound      = "tool_not_found"
	ErrCodeLLMError          = "llm_error"
	ErrCodeTimeout           = "timeout"
	ErrCodeAgentTimeout      = "agent_timeout"
	ErrCodeMaxIterations     = "max_iterations_exceeded"
	ErrCodeInvalidInput      = "invalid_input"
	ErrCodeResourceExhausted = "resource_exhausted"
	ErrCodeInvalidAction     = "invalid_action"
	ErrCodeToolExecution     = "tool_execution_failed"
	ErrCodeEventHandler      = "event_handler_error"
	ErrCodeStateTransition   = "state_transition_error"
	ErrCodeShutdown          = "shutdown_failed"
)

// NewAgentError creates a new AgentError.
func NewAgentError(op, agent, code string, err error) *AgentError {
	return &AgentError{
		Op:     op,
		Agent:  agent,
		Code:   code,
		Err:    err,
		Fields: make(map[string]any),
	}
}

// WithField adds a context field to the error.
func (e *AgentError) WithField(key string, value any) *AgentError {
	if e.Fields == nil {
		e.Fields = make(map[string]any)
	}
	e.Fields[key] = value
	return e
}

// ValidationError represents configuration validation errors.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// FactoryError represents errors that occur during agent creation.
type FactoryError struct {
	Config    any
	Err       error
	AgentType string
}

// Error implements the error interface.
func (e *FactoryError) Error() string {
	return fmt.Sprintf("failed to create agent of type '%s': %v", e.AgentType, e.Err)
}

// Unwrap returns the underlying error.
func (e *FactoryError) Unwrap() error {
	return e.Err
}

// NewFactoryError creates a new FactoryError.
func NewFactoryError(agentType string, config any, err error) *FactoryError {
	return &FactoryError{
		AgentType: agentType,
		Config:    config,
		Err:       err,
	}
}

// ExecutionError represents errors that occur during agent execution.
type ExecutionError struct {
	Err       error
	Agent     string
	Action    string
	Step      int
	Retryable bool
}

// Error implements the error interface.
func (e *ExecutionError) Error() string {
	return fmt.Sprintf("execution error for agent '%s' at step %d (action: %s): %v",
		e.Agent, e.Step, e.Action, e.Err)
}

// Unwrap returns the underlying error.
func (e *ExecutionError) Unwrap() error {
	return e.Err
}

// NewExecutionError creates a new ExecutionError.
func NewExecutionError(agent string, step int, action string, err error, retryable bool) *ExecutionError {
	return &ExecutionError{
		Agent:     agent,
		Step:      step,
		Action:    action,
		Err:       err,
		Retryable: retryable,
	}
}

// PlanningError represents errors that occur during the planning phase.
type PlanningError struct {
	Err        error
	Agent      string
	Suggestion string
	InputKeys  []string
}

// Error implements the error interface.
func (e *PlanningError) Error() string {
	msg := fmt.Sprintf("planning error for agent '%s': %v", e.Agent, e.Err)
	if len(e.InputKeys) > 0 {
		msg += fmt.Sprintf(" (expected input keys: %v)", e.InputKeys)
	}
	if e.Suggestion != "" {
		msg += ". Suggestion: " + e.Suggestion
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *PlanningError) Unwrap() error {
	return e.Err
}

// NewPlanningError creates a new PlanningError.
func NewPlanningError(agent string, inputKeys []string, err error) *PlanningError {
	return &PlanningError{
		Agent:     agent,
		InputKeys: inputKeys,
		Err:       err,
	}
}

// WithSuggestion adds a suggestion to help resolve the planning error.
func (e *PlanningError) WithSuggestion(suggestion string) *PlanningError {
	e.Suggestion = suggestion
	return e
}

// Common error variables for frequently used errors.
var (
	ErrAgentNotFound         = errors.New("agent not found")
	ErrInvalidConfig         = errors.New("invalid configuration")
	ErrToolNotAvailable      = errors.New("tool not available")
	ErrMaxIterationsExceeded = errors.New("maximum iterations exceeded")
	ErrContextCancelled      = errors.New("context canceled")
	ErrTimeout               = errors.New("operation timed out")
	ErrAgentTimeout          = errors.New("agent execution timed out")
	ErrResourceExhausted     = errors.New("resource exhausted")
	ErrInvalidAction         = errors.New("invalid action")
	ErrToolExecution         = errors.New("tool execution failed")
	ErrEventHandler          = errors.New("event handler error")
	ErrStateTransition       = errors.New("invalid state transition")
	ErrShutdown              = errors.New("shutdown failed")
)

// IsRetryable checks if an error is retryable.
func IsRetryable(err error) bool {
	var execErr *ExecutionError
	if errors.As(err, &execErr) {
		return execErr.Retryable
	}

	// Check for AgentError with retryable error codes
	var agentErr *AgentError
	if errors.As(err, &agentErr) {
		switch agentErr.Code {
		case ErrCodeTimeout, ErrCodeAgentTimeout, ErrCodeResourceExhausted,
			ErrCodeToolExecution, ErrCodeLLMError, ErrCodeExecution:
			return true
		case ErrCodeInvalidInput, ErrCodeConfigInvalid, ErrCodeInvalidAction,
			ErrCodeStateTransition, ErrCodeShutdown:
			return false
		}
	}

	// Check for common retryable error conditions
	if errors.Is(err, ErrTimeout) || errors.Is(err, ErrAgentTimeout) ||
		errors.Is(err, ErrResourceExhausted) || errors.Is(err, ErrToolExecution) {
		return true
	}

	return false
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	var valErr *ValidationError
	return errors.As(err, &valErr)
}

// IsFactoryError checks if an error is a factory error.
func IsFactoryError(err error) bool {
	var factErr *FactoryError
	return errors.As(err, &factErr)
}
