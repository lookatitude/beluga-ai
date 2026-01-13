// Package agents provides comprehensive tests for error types and error codes.
// T158: Add test cases for all error codes in errors_test.go
package agents

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Static error variables for testing (err113 compliance)
var (
	errOriginalError   = errors.New("original error")
	errExecutionFailed = errors.New("execution failed")
	errPlanningFailed  = errors.New("planning failed")
	errStreamingFailed = errors.New("streaming failed")
	errCreationFailed  = errors.New("creation failed")
	errRegularError    = errors.New("regular error")
)

// TestAgentError_Creation tests AgentError creation and basic functionality.
func TestAgentError_Creation(t *testing.T) {
	originalErr := errOriginalError
	agentErr := NewAgentError("test_operation", "test_agent", "test_code", originalErr)

	assert.Equal(t, "test_operation", agentErr.Op)
	assert.Equal(t, "test_agent", agentErr.Agent)
	assert.Equal(t, "test_code", agentErr.Code)
	assert.Equal(t, originalErr, agentErr.Err)
	assert.NotNil(t, agentErr.Fields)
	assert.ErrorIs(t, agentErr, originalErr)
}

// TestAgentError_ErrorString tests error string formatting.
func TestAgentError_ErrorString(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		agent    string
		code     string
		err      error
		contains []string
	}{
		{
			name:     "with agent name",
			op:       "execute",
			agent:    "test_agent",
			code:     ErrCodeExecution,
			err:      errExecutionFailed,
			contains: []string{"agent", "test_agent", "execute"},
		},
		{
			name:     "without agent name",
			op:       "plan",
			agent:    "",
			code:     ErrCodePlanning,
			err:      errPlanningFailed,
			contains: []string{"agent", "plan"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentErr := NewAgentError(tt.op, tt.agent, tt.code, tt.err)
			errStr := agentErr.Error()
			for _, substr := range tt.contains {
				assert.Contains(t, errStr, substr)
			}
		})
	}
}

// TestAgentError_WithField tests adding fields to errors.
func TestAgentError_WithField(t *testing.T) {
	agentErr := NewAgentError("test", "agent", "code", errOriginalError)

	agentErr.WithField("key1", "value1")
	agentErr.WithField("key2", 42)
	agentErr.WithField("key3", true)

	assert.Equal(t, "value1", agentErr.Fields["key1"])
	assert.Equal(t, 42, agentErr.Fields["key2"])
	assert.Equal(t, true, agentErr.Fields["key3"])
}

// TestAgentError_Unwrap tests error unwrapping.
func TestAgentError_Unwrap(t *testing.T) {
	originalErr := errOriginalError
	agentErr := NewAgentError("test", "agent", "code", originalErr)

	unwrapped := agentErr.Unwrap()
	assert.Equal(t, originalErr, unwrapped)
}

// TestAllErrorCodes tests all error code constants are defined and used correctly.
func TestAllErrorCodes(t *testing.T) {
	errorCodes := []struct {
		name string
		code string
	}{
		{"ConfigInvalid", ErrCodeConfigInvalid},
		{"Initialization", ErrCodeInitialization},
		{"Execution", ErrCodeExecution},
		{"Planning", ErrCodePlanning},
		{"ToolNotFound", ErrCodeToolNotFound},
		{"LLMError", ErrCodeLLMError},
		{"Timeout", ErrCodeTimeout},
		{"AgentTimeout", ErrCodeAgentTimeout},
		{"MaxIterations", ErrCodeMaxIterations},
		{"InvalidInput", ErrCodeInvalidInput},
		{"ResourceExhausted", ErrCodeResourceExhausted},
		{"InvalidAction", ErrCodeInvalidAction},
		{"ToolExecution", ErrCodeToolExecution},
		{"EventHandler", ErrCodeEventHandler},
		{"StateTransition", ErrCodeStateTransition},
		{"Shutdown", ErrCodeShutdown},
		{"StreamingNotSupported", ErrCodeStreamingNotSupported},
		{"StreamInterrupted", ErrCodeStreamInterrupted},
		{"StreamError", ErrCodeStreamError},
	}

	for _, tt := range errorCodes {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.code, "Error code should not be empty")
			// Create an error with this code to verify it works
			agentErr := NewAgentError("test", "agent", tt.code, errOriginalError)
			assert.Equal(t, tt.code, agentErr.Code)
		})
	}
}

// TestValidationError tests ValidationError creation and functionality.
func TestValidationError(t *testing.T) {
	valErr := NewValidationError("test_field", "test message")

	assert.Equal(t, "test_field", valErr.Field)
	assert.Equal(t, "test message", valErr.Message)
	assert.Contains(t, valErr.Error(), "validation error")
	assert.Contains(t, valErr.Error(), "test_field")
	assert.Contains(t, valErr.Error(), "test message")
}

// TestIsValidationError tests validation error detection.
func TestIsValidationError(t *testing.T) {
	valErr := NewValidationError("field", "message")
	assert.True(t, IsValidationError(valErr))

	agentErr := NewAgentError("op", "agent", "code", errOriginalError)
	assert.False(t, IsValidationError(agentErr))

	assert.False(t, IsValidationError(errRegularError))
}

// TestFactoryError tests FactoryError creation and functionality.
func TestFactoryError(t *testing.T) {
	originalErr := errCreationFailed
	config := map[string]any{"key": "value"}
	factoryErr := NewFactoryError("custom_agent", config, originalErr)

	assert.Equal(t, "custom_agent", factoryErr.AgentType)
	assert.Equal(t, config, factoryErr.Config)
	assert.Equal(t, originalErr, factoryErr.Err)
	assert.Contains(t, factoryErr.Error(), "failed to create agent")
	assert.Contains(t, factoryErr.Error(), "custom_agent")
	assert.ErrorIs(t, factoryErr, originalErr)
	assert.Equal(t, originalErr, factoryErr.Unwrap())
}

// TestIsFactoryError tests factory error detection.
func TestIsFactoryError(t *testing.T) {
	factoryErr := NewFactoryError("type", nil, errOriginalError)
	assert.True(t, IsFactoryError(factoryErr))

	agentErr := NewAgentError("op", "agent", "code", errOriginalError)
	assert.False(t, IsFactoryError(agentErr))

	assert.False(t, IsFactoryError(errRegularError))
}

// TestExecutionError tests ExecutionError creation and functionality.
func TestExecutionError(t *testing.T) {
	originalErr := errExecutionFailed
	execErr := NewExecutionError("test_agent", 5, "test_action", originalErr, true)

	assert.Equal(t, "test_agent", execErr.Agent)
	assert.Equal(t, 5, execErr.Step)
	assert.Equal(t, "test_action", execErr.Action)
	assert.Equal(t, originalErr, execErr.Err)
	assert.True(t, execErr.Retryable)
	assert.Contains(t, execErr.Error(), "execution error")
	assert.Contains(t, execErr.Error(), "test_agent")
	assert.Contains(t, execErr.Error(), "step 5")
	assert.ErrorIs(t, execErr, originalErr)
	assert.Equal(t, originalErr, execErr.Unwrap())
}

// TestPlanningError tests PlanningError creation and functionality.
func TestPlanningError(t *testing.T) {
	originalErr := errPlanningFailed
	inputKeys := []string{"input1", "input2"}
	planErr := NewPlanningError("test_agent", inputKeys, originalErr)

	assert.Equal(t, "test_agent", planErr.Agent)
	assert.Equal(t, inputKeys, planErr.InputKeys)
	assert.Equal(t, originalErr, planErr.Err)
	assert.Empty(t, planErr.Suggestion)
	assert.Contains(t, planErr.Error(), "planning error")
	assert.Contains(t, planErr.Error(), "test_agent")
	assert.Contains(t, planErr.Error(), "input1")
	assert.ErrorIs(t, planErr, originalErr)
	assert.Equal(t, originalErr, planErr.Unwrap())

	// Test WithSuggestion
	planErr.WithSuggestion("Check your inputs")
	assert.Equal(t, "Check your inputs", planErr.Suggestion)
	assert.Contains(t, planErr.Error(), "Suggestion: Check your inputs")
}

// TestStreamingError tests StreamingError creation and functionality.
func TestStreamingError(t *testing.T) {
	originalErr := errStreamingFailed
	streamErr := NewStreamingError("StreamExecute", "test_agent", ErrCodeStreamError, originalErr)

	assert.Equal(t, "StreamExecute", streamErr.Op)
	assert.Equal(t, "test_agent", streamErr.Agent)
	assert.Equal(t, ErrCodeStreamError, streamErr.Code)
	assert.Equal(t, originalErr, streamErr.Err)
	assert.NotNil(t, streamErr.Fields)
	assert.Contains(t, streamErr.Error(), "streaming error")
	assert.Contains(t, streamErr.Error(), "test_agent")
	assert.Contains(t, streamErr.Error(), "StreamExecute")
	assert.Contains(t, streamErr.Error(), ErrCodeStreamError)
	assert.ErrorIs(t, streamErr, originalErr)
	assert.Equal(t, originalErr, streamErr.Unwrap())
}

// TestStreamingError_WithField tests adding fields to streaming errors.
func TestStreamingError_WithField(t *testing.T) {
	streamErr := NewStreamingError("test", "agent", ErrCodeStreamError, errOriginalError)

	streamErr.WithField("chunk_count", 10)
	streamErr.WithField("duration_ms", 500)

	assert.Equal(t, 10, streamErr.Fields["chunk_count"])
	assert.Equal(t, 500, streamErr.Fields["duration_ms"])
}

// TestWrapStreamingError tests wrapping errors as StreamingError.
func TestWrapStreamingError(t *testing.T) {
	originalErr := errOriginalError
	wrappedErr := WrapStreamingError("test", "agent", ErrCodeStreamInterrupted, originalErr)

	assert.IsType(t, &StreamingError{}, wrappedErr)
	assert.Equal(t, originalErr, wrappedErr.Err)
	assert.Equal(t, ErrCodeStreamInterrupted, wrappedErr.Code)
}

// TestIsStreamingError tests streaming error detection.
func TestIsStreamingError(t *testing.T) {
	streamErr := NewStreamingError("test", "agent", ErrCodeStreamError, errOriginalError)
	assert.True(t, IsStreamingError(streamErr))

	agentErr := NewAgentError("op", "agent", "code", errOriginalError)
	assert.False(t, IsStreamingError(agentErr))

	assert.False(t, IsStreamingError(errRegularError))
}

// TestStreamingErrorCodes tests all streaming error codes.
func TestStreamingErrorCodes(t *testing.T) {
	codes := []string{
		ErrCodeStreamingNotSupported,
		ErrCodeStreamInterrupted,
		ErrCodeStreamError,
	}

	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			streamErr := NewStreamingError("test", "agent", code, errOriginalError)
			assert.Equal(t, code, streamErr.Code)
		})
	}
}

// TestIsRetryable tests retryable error detection for different error types.
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		err       error
		name      string
		retryable bool
	}{
		// ExecutionError with Retryable=true
		{
			name:      "execution error retryable",
			err:       NewExecutionError("agent", 1, "action", errOriginalError, true),
			retryable: true,
		},
		// ExecutionError with Retryable=false
		{
			name:      "execution error non-retryable",
			err:       NewExecutionError("agent", 1, "action", errOriginalError, false),
			retryable: false,
		},
		// AgentError with retryable codes
		{
			name:      "timeout error",
			err:       NewAgentError("op", "agent", ErrCodeTimeout, errOriginalError),
			retryable: true,
		},
		{
			name:      "agent timeout error",
			err:       NewAgentError("op", "agent", ErrCodeAgentTimeout, errOriginalError),
			retryable: true,
		},
		{
			name:      "resource exhausted error",
			err:       NewAgentError("op", "agent", ErrCodeResourceExhausted, errOriginalError),
			retryable: true,
		},
		{
			name:      "tool execution error",
			err:       NewAgentError("op", "agent", ErrCodeToolExecution, errOriginalError),
			retryable: true,
		},
		{
			name:      "LLM error",
			err:       NewAgentError("op", "agent", ErrCodeLLMError, errOriginalError),
			retryable: true,
		},
		{
			name:      "execution error code",
			err:       NewAgentError("op", "agent", ErrCodeExecution, errOriginalError),
			retryable: true,
		},
		// AgentError with non-retryable codes
		{
			name:      "invalid input error",
			err:       NewAgentError("op", "agent", ErrCodeInvalidInput, errOriginalError),
			retryable: false,
		},
		{
			name:      "config invalid error",
			err:       NewAgentError("op", "agent", ErrCodeConfigInvalid, errOriginalError),
			retryable: false,
		},
		{
			name:      "invalid action error",
			err:       NewAgentError("op", "agent", ErrCodeInvalidAction, errOriginalError),
			retryable: false,
		},
		{
			name:      "state transition error",
			err:       NewAgentError("op", "agent", ErrCodeStateTransition, errOriginalError),
			retryable: false,
		},
		{
			name:      "shutdown error",
			err:       NewAgentError("op", "agent", ErrCodeShutdown, errOriginalError),
			retryable: false,
		},
		// Common error variables
		{
			name:      "ErrTimeout",
			err:       ErrTimeout,
			retryable: true,
		},
		{
			name:      "ErrAgentTimeout",
			err:       ErrAgentTimeout,
			retryable: true,
		},
		{
			name:      "ErrResourceExhausted",
			err:       ErrResourceExhausted,
			retryable: true,
		},
		{
			name:      "ErrToolExecution",
			err:       ErrToolExecution,
			retryable: true,
		},
		// Non-retryable cases
		{
			name:      "regular error",
			err:       errRegularError,
			retryable: false,
		},
		{
			name:      "validation error",
			err:       NewValidationError("field", "message"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			assert.Equal(t, tt.retryable, result, "Expected retryable=%v for error: %v", tt.retryable, tt.err)
		})
	}
}

// TestCommonErrorVariables tests all common error variables are defined.
func TestCommonErrorVariables(t *testing.T) {
	errorVars := []struct {
		err  error
		name string
	}{
		{ErrAgentNotFound, "ErrAgentNotFound"},
		{ErrInvalidConfig, "ErrInvalidConfig"},
		{ErrToolNotAvailable, "ErrToolNotAvailable"},
		{ErrMaxIterationsExceeded, "ErrMaxIterationsExceeded"},
		{ErrContextCancelled, "ErrContextCancelled"},
		{ErrTimeout, "ErrTimeout"},
		{ErrAgentTimeout, "ErrAgentTimeout"},
		{ErrResourceExhausted, "ErrResourceExhausted"},
		{ErrInvalidAction, "ErrInvalidAction"},
		{ErrToolExecution, "ErrToolExecution"},
		{ErrEventHandler, "ErrEventHandler"},
		{ErrStateTransition, "ErrStateTransition"},
		{ErrShutdown, "ErrShutdown"},
	}

	for _, tt := range errorVars {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, tt.err, "Error variable should not be nil")
			assert.NotEmpty(t, tt.err.Error(), "Error should have a message")
		})
	}
}

// TestErrorWrapping tests that errors properly wrap underlying errors.
func TestErrorWrapping(t *testing.T) {
	originalErr := errOriginalError

	// Test AgentError wrapping
	agentErr := NewAgentError("op", "agent", "code", originalErr)
	assert.ErrorIs(t, agentErr, originalErr)
	assert.Equal(t, originalErr, agentErr.Unwrap())

	// Test FactoryError wrapping
	factoryErr := NewFactoryError("type", nil, originalErr)
	assert.ErrorIs(t, factoryErr, originalErr)
	assert.Equal(t, originalErr, factoryErr.Unwrap())

	// Test ExecutionError wrapping
	execErr := NewExecutionError("agent", 1, "action", originalErr, false)
	assert.ErrorIs(t, execErr, originalErr)
	assert.Equal(t, originalErr, execErr.Unwrap())

	// Test PlanningError wrapping
	planErr := NewPlanningError("agent", nil, originalErr)
	assert.ErrorIs(t, planErr, originalErr)
	assert.Equal(t, originalErr, planErr.Unwrap())

	// Test StreamingError wrapping
	streamErr := NewStreamingError("op", "agent", "code", originalErr)
	assert.ErrorIs(t, streamErr, originalErr)
	assert.Equal(t, originalErr, streamErr.Unwrap())
}

// TestStreamingError_ErrorStringVariations tests error string formatting with and without agent.
func TestStreamingError_ErrorStringVariations(t *testing.T) {
	err := errOriginalError

	t.Run("with agent", func(t *testing.T) {
		streamErr := NewStreamingError("test_op", "test_agent", ErrCodeStreamError, err)
		errStr := streamErr.Error()
		assert.Contains(t, errStr, "test_agent")
		assert.Contains(t, errStr, "test_op")
		assert.Contains(t, errStr, ErrCodeStreamError)
	})

	t.Run("without agent", func(t *testing.T) {
		streamErr := NewStreamingError("test_op", "", ErrCodeStreamError, err)
		errStr := streamErr.Error()
		assert.NotContains(t, errStr, "agent test_agent")
		assert.Contains(t, errStr, "test_op")
		assert.Contains(t, errStr, ErrCodeStreamError)
	})
}
