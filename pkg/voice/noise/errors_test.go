package noise

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoiseCancellationError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *NoiseCancellationError
		wantMsg string
	}{
		{
			name:    "error with message",
			err:     NewNoiseCancellationErrorWithMessage("Process", ErrCodeInternalError, "processing failed", nil),
			wantMsg: "noise Process: processing failed (code: internal_error)",
		},
		{
			name:    "error with underlying error",
			err:     NewNoiseCancellationError("Process", ErrCodeTimeout, errors.New("context deadline exceeded")),
			wantMsg: "noise Process: context deadline exceeded (code: timeout)",
		},
		{
			name:    "error without message or underlying error",
			err:     NewNoiseCancellationError("Process", ErrCodeInternalError, nil),
			wantMsg: "noise Process: unknown error (code: internal_error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestNoiseCancellationError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := NewNoiseCancellationError("Process", ErrCodeInternalError, underlying)
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
			err:  NewNoiseCancellationError("Process", ErrCodeTimeout, nil),
			want: true,
		},
		{
			name: "retryable processing error",
			err:  NewNoiseCancellationError("Process", ErrCodeProcessingError, nil),
			want: true,
		},
		{
			name: "non-retryable invalid config error",
			err:  NewNoiseCancellationError("Process", ErrCodeInvalidConfig, nil),
			want: false,
		},
		{
			name: "non-retryable internal error",
			err:  NewNoiseCancellationError("Process", ErrCodeInternalError, nil),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non-NoiseCancellation error",
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
