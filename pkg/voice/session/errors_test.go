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
