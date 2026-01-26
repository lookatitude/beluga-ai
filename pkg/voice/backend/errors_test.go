package backend

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendError(t *testing.T) {
	tests := []struct {
		name        string
		err         *BackendError
		wantMessage string
	}{
		{
			name: "error with message",
			err: &BackendError{
				Op:      "CreateSession",
				Code:    ErrCodeInvalidConfig,
				Message: "invalid configuration",
			},
			wantMessage: "voice/backend CreateSession: invalid configuration (code: invalid_config)",
		},
		{
			name: "error with underlying error",
			err: &BackendError{
				Op:   "CreateSession",
				Code: ErrCodeConnectionFailed,
				Err:  errors.New("connection refused"),
			},
			wantMessage: "voice/backend CreateSession: connection refused (code: connection_failed)",
		},
		{
			name: "error without message or underlying error",
			err: &BackendError{
				Op:   "CreateSession",
				Code: ErrCodeTimeout,
			},
			wantMessage: "voice/backend CreateSession: unknown error (code: timeout)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMessage, tt.err.Error())
		})
	}
}

func TestBackendErrorUnwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &BackendError{
		Op:   "test",
		Code: ErrCodePipelineError,
		Err:  underlying,
	}

	assert.Equal(t, underlying, err.Unwrap())
}

func TestNewBackendError(t *testing.T) {
	underlying := errors.New("test error")
	err := NewBackendError("TestOp", ErrCodeInvalidConfig, underlying)

	require.NotNil(t, err)
	assert.Equal(t, "TestOp", err.Op)
	assert.Equal(t, ErrCodeInvalidConfig, err.Code)
	assert.Equal(t, underlying, err.Err)
	assert.Empty(t, err.Message)
}

func TestNewBackendErrorWithMessage(t *testing.T) {
	underlying := errors.New("test error")
	err := NewBackendErrorWithMessage("TestOp", ErrCodeProviderNotFound, "provider not found", underlying)

	require.NotNil(t, err)
	assert.Equal(t, "TestOp", err.Op)
	assert.Equal(t, ErrCodeProviderNotFound, err.Code)
	assert.Equal(t, "provider not found", err.Message)
	assert.Equal(t, underlying, err.Err)
}

func TestNewBackendErrorWithDetails(t *testing.T) {
	underlying := errors.New("test error")
	details := map[string]any{
		"session_id": "test-123",
		"provider":   "livekit",
	}
	err := NewBackendErrorWithDetails("TestOp", ErrCodeSessionNotFound, "session not found", underlying, details)

	require.NotNil(t, err)
	assert.Equal(t, "TestOp", err.Op)
	assert.Equal(t, ErrCodeSessionNotFound, err.Code)
	assert.Equal(t, "session not found", err.Message)
	assert.Equal(t, underlying, err.Err)
	assert.Equal(t, details, err.Details)
}

func TestWrapError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := WrapError("TestOp", nil)
		assert.Nil(t, result)
	})

	t.Run("regular error", func(t *testing.T) {
		err := errors.New("some error")
		result := WrapError("TestOp", err)

		require.NotNil(t, result)
		backendErr, ok := result.(*BackendError)
		require.True(t, ok)
		assert.Equal(t, "TestOp", backendErr.Op)
		assert.Equal(t, ErrCodePipelineError, backendErr.Code)
	})

	t.Run("already BackendError", func(t *testing.T) {
		original := NewBackendError("OriginalOp", ErrCodeConnectionFailed, errors.New("connection error"))
		result := WrapError("NewOp", original)

		require.NotNil(t, result)
		backendErr, ok := result.(*BackendError)
		require.True(t, ok)
		assert.Equal(t, "NewOp", backendErr.Op)
		assert.Equal(t, ErrCodeConnectionFailed, backendErr.Code)
	})
}

func TestIsError(t *testing.T) {
	t.Run("BackendError", func(t *testing.T) {
		err := NewBackendError("Test", ErrCodeInvalidConfig, nil)
		assert.True(t, IsError(err))
	})

	t.Run("regular error", func(t *testing.T) {
		err := errors.New("regular error")
		assert.False(t, IsError(err))
	})

	t.Run("wrapped BackendError", func(t *testing.T) {
		backendErr := NewBackendError("Test", ErrCodeInvalidConfig, nil)
		wrapped := errors.Join(errors.New("wrapper"), backendErr)
		assert.True(t, IsError(wrapped))
	})
}

func TestAsError(t *testing.T) {
	t.Run("BackendError", func(t *testing.T) {
		err := NewBackendError("Test", ErrCodeInvalidConfig, nil)
		result := AsError(err)
		require.NotNil(t, result)
		assert.Equal(t, "Test", result.Op)
	})

	t.Run("regular error", func(t *testing.T) {
		err := errors.New("regular error")
		result := AsError(err)
		assert.Nil(t, result)
	})
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "connection timeout - retryable",
			err:       NewBackendError("Test", ErrCodeConnectionTimeout, nil),
			retryable: true,
		},
		{
			name:      "rate limit exceeded - retryable",
			err:       NewBackendError("Test", ErrCodeRateLimitExceeded, nil),
			retryable: true,
		},
		{
			name:      "connection failed - retryable",
			err:       NewBackendError("Test", ErrCodeConnectionFailed, nil),
			retryable: true,
		},
		{
			name:      "invalid config - not retryable",
			err:       NewBackendError("Test", ErrCodeInvalidConfig, nil),
			retryable: false,
		},
		{
			name:      "authentication failed - not retryable",
			err:       NewBackendError("Test", ErrCodeAuthenticationFailed, nil),
			retryable: false,
		},
		{
			name:      "regular error - not retryable",
			err:       errors.New("regular error"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.retryable, IsRetryableError(tt.err))
		})
	}
}

func TestErrorCodes(t *testing.T) {
	// Verify all error codes are defined and unique
	codes := []string{
		ErrCodeInvalidConfig,
		ErrCodeProviderNotFound,
		ErrCodeConnectionFailed,
		ErrCodeConnectionTimeout,
		ErrCodeSessionNotFound,
		ErrCodeSessionLimitExceeded,
		ErrCodeRateLimitExceeded,
		ErrCodeAuthenticationFailed,
		ErrCodeAuthorizationFailed,
		ErrCodePipelineError,
		ErrCodeAgentError,
		ErrCodeTimeout,
		ErrCodeContextCanceled,
		ErrCodeInvalidFormat,
		ErrCodeConversionFailed,
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		assert.NotEmpty(t, code, "error code should not be empty")
		assert.False(t, seen[code], "duplicate error code: %s", code)
		seen[code] = true
	}
}
