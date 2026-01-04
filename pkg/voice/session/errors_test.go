package session

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *SessionError
		wantMsg string
	}{
		{
			name:    "error with message",
			err:     NewSessionErrorWithMessage("Start", ErrCodeInternalError, "session start failed", nil),
			wantMsg: "session Start: session start failed (code: internal_error)",
		},
		{
			name:    "error with underlying error",
			err:     NewSessionError("Start", ErrCodeTimeout, errors.New("context deadline exceeded")),
			wantMsg: "session Start: context deadline exceeded (code: timeout)",
		},
		{
			name:    "error without message or underlying error",
			err:     NewSessionError("Start", ErrCodeInternalError, nil),
			wantMsg: "session Start: unknown error (code: internal_error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestSessionError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := NewSessionError("Start", ErrCodeInternalError, underlying)
	assert.Equal(t, underlying, err.Unwrap())
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		err  error
		name string
		want bool
	}{
		{
			name: "retryable timeout error",
			err:  NewSessionError("Start", ErrCodeTimeout, nil),
			want: true,
		},
		{
			name: "retryable context timeout error",
			err:  NewSessionError("Start", ErrCodeContextTimeout, nil),
			want: true,
		},
		{
			name: "non-retryable invalid config error",
			err:  NewSessionError("Start", ErrCodeInvalidConfig, nil),
			want: false,
		},
		{
			name: "non-retryable internal error",
			err:  NewSessionError("Start", ErrCodeInternalError, nil),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non-Session error",
			err:  errors.New("generic error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRetryableError(tt.err))
		})
	}
}

// TestAllErrorCodes tests that all error codes can be created and formatted correctly.
func TestAllErrorCodes(t *testing.T) {
	errorCodes := []struct {
		code string
		name string
	}{
		{ErrCodeInvalidConfig, "invalid_config"},
		{ErrCodeInternalError, "internal_error"},
		{ErrCodeInvalidState, "invalid_state"},
		{ErrCodeTimeout, "timeout"},
		{ErrCodeSessionNotFound, "session_not_found"},
		{ErrCodeSessionAlreadyActive, "session_already_active"},
		{ErrCodeSessionNotActive, "session_not_active"},
		{ErrCodeSessionExpired, "session_expired"},
		{ErrCodeContextCanceled, "context_canceled"},
		{ErrCodeContextTimeout, "context_timeout"},
		{ErrCodeAgentNotSet, "agent_not_set"},
		{ErrCodeAgentInvalid, "agent_invalid"},
		{ErrCodeStreamError, "stream_error"},
		{ErrCodeContextError, "context_error"},
		{ErrCodeInterruptionError, "interruption_error"},
	}

	for _, tc := range errorCodes {
		t.Run(tc.name, func(t *testing.T) {
			err := NewSessionError("TestOp", tc.code, nil)
			assert.NotNil(t, err)
			assert.Equal(t, tc.code, err.Code)
			assert.Equal(t, "TestOp", err.Op)
			assert.Contains(t, err.Error(), tc.code)
		})
	}
}

// TestNewSessionErrorWithDetails tests error creation with details.
func TestNewSessionErrorWithDetails(t *testing.T) {
	details := map[string]any{
		"session_id":  "test-123",
		"retry_count": 3,
		"timestamp":   "2024-01-01T00:00:00Z",
	}

	err := NewSessionErrorWithDetails("Start", ErrCodeInternalError, "test error", errors.New("underlying"), details)
	assert.NotNil(t, err)
	assert.Equal(t, "Start", err.Op)
	assert.Equal(t, ErrCodeInternalError, err.Code)
	assert.Equal(t, "test error", err.Message)
	assert.Equal(t, details, err.Details)
	assert.Error(t, err.Err)
}

// TestNewAgentIntegrationError tests agent integration error creation.
func TestNewAgentIntegrationError(t *testing.T) {
	underlyingErr := errors.New("agent error")
	err := NewAgentIntegrationError("StreamExecute", ErrCodeAgentNotSet, underlyingErr)
	assert.NotNil(t, err)
	assert.Equal(t, "StreamExecute", err.Op)
	assert.Equal(t, ErrCodeAgentNotSet, err.Code)
	assert.Equal(t, underlyingErr, err.Err)
}

// TestWrapAgentIntegrationError tests wrapping errors for agent integration.
func TestWrapAgentIntegrationError(t *testing.T) {
	t.Run("wrap non-nil error", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		err := WrapAgentIntegrationError("StartStreaming", ErrCodeStreamError, underlyingErr)
		assert.NotNil(t, err)
		assert.Equal(t, "StartStreaming", err.Op)
		assert.Equal(t, ErrCodeStreamError, err.Code)
		assert.Equal(t, underlyingErr, err.Err)
	})

	t.Run("wrap nil error returns nil", func(t *testing.T) {
		err := WrapAgentIntegrationError("StartStreaming", ErrCodeStreamError, nil)
		assert.Nil(t, err)
	})
}

// TestSessionError_ErrorString tests different error string formats.
func TestSessionError_ErrorString(t *testing.T) {
	tests := []struct {
		name     string
		err      *SessionError
		contains []string
	}{
		{
			name:     "error with message",
			err:      NewSessionErrorWithMessage("Start", ErrCodeTimeout, "operation timed out", nil),
			contains: []string{"session Start", "operation timed out", "timeout"},
		},
		{
			name:     "error with underlying error",
			err:      NewSessionError("ProcessAudio", ErrCodeInternalError, errors.New("internal failure")),
			contains: []string{"session ProcessAudio", "internal failure", "internal_error"},
		},
		{
			name:     "error without message or underlying error",
			err:      NewSessionError("Stop", ErrCodeInvalidState, nil),
			contains: []string{"session Stop", "unknown error", "invalid_state"},
		},
		{
			name: "error with details",
			err: NewSessionErrorWithDetails("Start", ErrCodeSessionNotFound, "session not found",
				errors.New("not found"), map[string]any{"session_id": "test-123"}),
			contains: []string{"session Start", "session not found", "session_not_found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorStr := tt.err.Error()
			for _, substr := range tt.contains {
				assert.Contains(t, errorStr, substr, "Error string should contain: %s", substr)
			}
		})
	}
}

// TestSessionError_UnwrapAll tests unwrapping nested errors.
func TestSessionError_UnwrapAll(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := errors.New("wrapped: " + originalErr.Error())
	sessionErr := NewSessionError("TestOp", ErrCodeInternalError, wrappedErr)

	unwrapped := sessionErr.Unwrap()
	assert.Error(t, unwrapped)
	assert.Equal(t, wrappedErr, unwrapped)
}

// TestAgentIntegrationErrorCodes tests all agent integration error codes.
func TestAgentIntegrationErrorCodes(t *testing.T) {
	agentErrorCodes := []struct {
		code string
		name string
	}{
		{ErrCodeAgentNotSet, "agent_not_set"},
		{ErrCodeAgentInvalid, "agent_invalid"},
		{ErrCodeStreamError, "stream_error"},
		{ErrCodeContextError, "context_error"},
		{ErrCodeInterruptionError, "interruption_error"},
	}

	for _, tc := range agentErrorCodes {
		t.Run(tc.name, func(t *testing.T) {
			err := NewAgentIntegrationError("TestOp", tc.code, errors.New("test error"))
			assert.NotNil(t, err)
			assert.Equal(t, tc.code, err.Code)
			assert.Contains(t, err.Error(), tc.code)
		})
	}
}

// TestSessionLifecycleErrorCodes tests all session lifecycle error codes.
func TestSessionLifecycleErrorCodes(t *testing.T) {
	lifecycleErrorCodes := []struct {
		code string
		name string
	}{
		{ErrCodeSessionNotFound, "session_not_found"},
		{ErrCodeSessionAlreadyActive, "session_already_active"},
		{ErrCodeSessionNotActive, "session_not_active"},
		{ErrCodeSessionExpired, "session_expired"},
	}

	for _, tc := range lifecycleErrorCodes {
		t.Run(tc.name, func(t *testing.T) {
			err := NewSessionError("TestOp", tc.code, nil)
			assert.NotNil(t, err)
			assert.Equal(t, tc.code, err.Code)
			assert.Contains(t, err.Error(), tc.code)
			assert.False(t, IsRetryableError(err), "%s should not be retryable", tc.code)
		})
	}
}

// TestContextErrorCodes tests all context error codes.
func TestContextErrorCodes(t *testing.T) {
	t.Run("context_canceled", func(t *testing.T) {
		err := NewSessionError("TestOp", ErrCodeContextCanceled, errors.New("context canceled"))
		assert.NotNil(t, err)
		assert.Equal(t, ErrCodeContextCanceled, err.Code)
		assert.False(t, IsRetryableError(err))
	})

	t.Run("context_timeout is retryable", func(t *testing.T) {
		err := NewSessionError("TestOp", ErrCodeContextTimeout, nil)
		assert.NotNil(t, err)
		assert.Equal(t, ErrCodeContextTimeout, err.Code)
		assert.True(t, IsRetryableError(err))
	})
}

// TestGeneralErrorCodes tests all general error codes.
func TestGeneralErrorCodes(t *testing.T) {
	generalErrorCodes := []struct {
		code      string
		name      string
		retryable bool
	}{
		{ErrCodeInvalidConfig, "invalid_config", false},
		{ErrCodeInternalError, "internal_error", false},
		{ErrCodeInvalidState, "invalid_state", false},
		{ErrCodeTimeout, "timeout", true},
	}

	for _, tc := range generalErrorCodes {
		t.Run(tc.name, func(t *testing.T) {
			err := NewSessionError("TestOp", tc.code, nil)
			assert.NotNil(t, err)
			assert.Equal(t, tc.code, err.Code)
			assert.Equal(t, tc.retryable, IsRetryableError(err), "%s retryable status should match", tc.code)
		})
	}
}
