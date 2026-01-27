package turndetection

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTurnDetectionError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *TurnDetectionError
		wantMsg string
	}{
		{
			name:    "error with message",
			err:     NewTurnDetectionErrorWithMessage("DetectTurn", ErrCodeInternalError, "detection failed", nil),
			wantMsg: "turndetection DetectTurn: detection failed (code: internal_error)",
		},
		{
			name:    "error with underlying error",
			err:     NewTurnDetectionError("DetectTurn", ErrCodeTimeout, errors.New("context deadline exceeded")),
			wantMsg: "turndetection DetectTurn: context deadline exceeded (code: timeout)",
		},
		{
			name:    "error without message or underlying error",
			err:     NewTurnDetectionError("DetectTurn", ErrCodeInternalError, nil),
			wantMsg: "turndetection DetectTurn: unknown error (code: internal_error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestTurnDetectionError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := NewTurnDetectionError("DetectTurn", ErrCodeInternalError, underlying)
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
			err:  NewTurnDetectionError("DetectTurn", ErrCodeTimeout, nil),
			want: true,
		},
		{
			name: "retryable processing error",
			err:  NewTurnDetectionError("DetectTurn", ErrCodeProcessingError, nil),
			want: true,
		},
		{
			name: "non-retryable invalid config error",
			err:  NewTurnDetectionError("DetectTurn", ErrCodeInvalidConfig, nil),
			want: false,
		},
		{
			name: "non-retryable internal error",
			err:  NewTurnDetectionError("DetectTurn", ErrCodeInternalError, nil),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non-TurnDetection error",
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
