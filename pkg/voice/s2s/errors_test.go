package s2s

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestS2SError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *S2SError
		wantMsg string
	}{
		{
			name:    "error with message",
			err:     NewS2SErrorWithMessage("Process", ErrCodeNetworkError, "network failed", nil),
			wantMsg: "s2s Process: network failed (code: network_error)",
		},
		{
			name:    "error with underlying error",
			err:     NewS2SError("Process", ErrCodeTimeout, errors.New("context deadline exceeded")),
			wantMsg: "s2s Process: context deadline exceeded (code: timeout)",
		},
		{
			name:    "error without message or underlying error",
			err:     NewS2SError("Process", ErrCodeInternalError, nil),
			wantMsg: "s2s Process: unknown error (code: internal_error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestS2SError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := NewS2SError("Process", ErrCodeNetworkError, underlying)
	assert.Equal(t, underlying, err.Unwrap())
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		err  error
		name string
		want bool
	}{
		{
			name: "retryable network error",
			err:  NewS2SError("Process", ErrCodeNetworkError, nil),
			want: true,
		},
		{
			name: "retryable timeout error",
			err:  NewS2SError("Process", ErrCodeTimeout, nil),
			want: true,
		},
		{
			name: "retryable rate limit error",
			err:  NewS2SError("Process", ErrCodeRateLimit, nil),
			want: true,
		},
		{
			name: "non-retryable authentication error",
			err:  NewS2SError("Process", ErrCodeAuthentication, nil),
			want: false,
		},
		{
			name: "non-retryable invalid config error",
			err:  NewS2SError("Process", ErrCodeInvalidConfig, nil),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non-S2S error",
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
		statusCode int
		name       string
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
			name:       "rate limit",
			statusCode: http.StatusTooManyRequests,
			wantCode:   ErrCodeRateLimit,
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			wantCode:   ErrCodeInternalError,
		},
		{
			name:       "bad gateway",
			statusCode: http.StatusBadGateway,
			wantCode:   ErrCodeInternalError,
		},
		{
			name:       "service unavailable",
			statusCode: http.StatusServiceUnavailable,
			wantCode:   ErrCodeInternalError,
		},
		{
			name:       "unknown status code",
			statusCode: 999,
			wantCode:   ErrCodeNetworkError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrorFromHTTPStatus("TestOp", tt.statusCode, nil)
			assert.Equal(t, tt.wantCode, err.Code)
			assert.Equal(t, "TestOp", err.Op)
		})
	}
}
