package vad

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVADError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *VADError
		wantMsg string
	}{
		{
			name:    "error with message",
			err:     NewVADErrorWithMessage("Process", ErrCodeInternalError, "processing failed", nil),
			wantMsg: "vad Process: processing failed (code: internal_error)",
		},
		{
			name:    "error with underlying error",
			err:     NewVADError("Process", ErrCodeTimeout, errors.New("context deadline exceeded")),
			wantMsg: "vad Process: context deadline exceeded (code: timeout)",
		},
		{
			name:    "error without message or underlying error",
			err:     NewVADError("Process", ErrCodeInternalError, nil),
			wantMsg: "vad Process: unknown error (code: internal_error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestVADError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := NewVADError("Process", ErrCodeInternalError, underlying)
	assert.Equal(t, underlying, err.Unwrap())
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "retryable timeout error",
			err:  NewVADError("Process", ErrCodeTimeout, nil),
			want: true,
		},
		{
			name: "retryable processing error",
			err:  NewVADError("Process", ErrCodeProcessingError, nil),
			want: true,
		},
		{
			name: "non-retryable invalid config error",
			err:  NewVADError("Process", ErrCodeInvalidConfig, nil),
			want: false,
		},
		{
			name: "non-retryable internal error",
			err:  NewVADError("Process", ErrCodeInternalError, nil),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non-VAD error",
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
