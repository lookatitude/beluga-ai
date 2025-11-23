package tts

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTTSError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *TTSError
		wantMsg string
	}{
		{
			name:    "error with message",
			err:     NewTTSErrorWithMessage("GenerateSpeech", ErrCodeNetworkError, "network failed", nil),
			wantMsg: "tts GenerateSpeech: network failed (code: network_error)",
		},
		{
			name:    "error with underlying error",
			err:     NewTTSError("GenerateSpeech", ErrCodeTimeout, errors.New("context deadline exceeded")),
			wantMsg: "tts GenerateSpeech: context deadline exceeded (code: timeout)",
		},
		{
			name:    "error without message or underlying error",
			err:     NewTTSError("GenerateSpeech", ErrCodeInternalError, nil),
			wantMsg: "tts GenerateSpeech: unknown error (code: internal_error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestTTSError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := NewTTSError("GenerateSpeech", ErrCodeNetworkError, underlying)
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
			err:  NewTTSError("GenerateSpeech", ErrCodeNetworkError, nil),
			want: true,
		},
		{
			name: "retryable timeout error",
			err:  NewTTSError("GenerateSpeech", ErrCodeTimeout, nil),
			want: true,
		},
		{
			name: "retryable rate limit error",
			err:  NewTTSError("GenerateSpeech", ErrCodeRateLimit, nil),
			want: true,
		},
		{
			name: "non-retryable authentication error",
			err:  NewTTSError("GenerateSpeech", ErrCodeAuthentication, nil),
			want: false,
		},
		{
			name: "non-retryable invalid config error",
			err:  NewTTSError("GenerateSpeech", ErrCodeInvalidConfig, nil),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non-TTS error",
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
