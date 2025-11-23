package stt

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSTTError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *STTError
		wantMsg string
	}{
		{
			name:    "error with message",
			err:     NewSTTErrorWithMessage("Transcribe", ErrCodeNetworkError, "network failed", nil),
			wantMsg: "stt Transcribe: network failed (code: network_error)",
		},
		{
			name:    "error with underlying error",
			err:     NewSTTError("Transcribe", ErrCodeTimeout, errors.New("context deadline exceeded")),
			wantMsg: "stt Transcribe: context deadline exceeded (code: timeout)",
		},
		{
			name:    "error without message or underlying error",
			err:     NewSTTError("Transcribe", ErrCodeInternalError, nil),
			wantMsg: "stt Transcribe: unknown error (code: internal_error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestSTTError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := NewSTTError("Transcribe", ErrCodeNetworkError, underlying)
	assert.Equal(t, underlying, err.Unwrap())
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "retryable network error",
			err:  NewSTTError("Transcribe", ErrCodeNetworkError, nil),
			want: true,
		},
		{
			name: "retryable timeout error",
			err:  NewSTTError("Transcribe", ErrCodeTimeout, nil),
			want: true,
		},
		{
			name: "retryable rate limit error",
			err:  NewSTTError("Transcribe", ErrCodeRateLimit, nil),
			want: true,
		},
		{
			name: "non-retryable authentication error",
			err:  NewSTTError("Transcribe", ErrCodeAuthentication, nil),
			want: false,
		},
		{
			name: "non-retryable invalid config error",
			err:  NewSTTError("Transcribe", ErrCodeInvalidConfig, nil),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non-STT error",
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

func TestErrorFromHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantCode   string
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			wantCode:   ErrCodeInvalidRequest,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantCode:   ErrCodeAuthentication,
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			wantCode:   ErrCodeAuthorization,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantCode:   ErrCodeNotFound,
		},
		{
			name:       "too many requests",
			statusCode: http.StatusTooManyRequests,
			wantCode:   ErrCodeRateLimit,
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			wantCode:   ErrCodeInternalError,
		},
		{
			name:       "unknown status",
			statusCode: 999,
			wantCode:   ErrCodeNetworkError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrorFromHTTPStatus("Test", tt.statusCode, nil)
			assert.Equal(t, tt.wantCode, err.Code)
			assert.Equal(t, "Test", err.Op)
		})
	}
}
