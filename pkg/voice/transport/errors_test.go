package transport

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransportError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *TransportError
		wantMsg string
	}{
		{
			name:    "error with message",
			err:     NewTransportErrorWithMessage("Connect", ErrCodeConnectionFailed, "connection failed", nil),
			wantMsg: "transport Connect: connection failed (code: connection_failed)",
		},
		{
			name:    "error with underlying error",
			err:     NewTransportError("Connect", ErrCodeTimeout, errors.New("context deadline exceeded")),
			wantMsg: "transport Connect: context deadline exceeded (code: timeout)",
		},
		{
			name:    "error without message or underlying error",
			err:     NewTransportError("Connect", ErrCodeInternalError, nil),
			wantMsg: "transport Connect: unknown error (code: internal_error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestTransportError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := NewTransportError("Connect", ErrCodeConnectionFailed, underlying)
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
			err:  NewTransportError("Connect", ErrCodeTimeout, nil),
			want: true,
		},
		{
			name: "retryable network error",
			err:  NewTransportError("Connect", ErrCodeNetworkError, nil),
			want: true,
		},
		{
			name: "retryable connection failed error",
			err:  NewTransportError("Connect", ErrCodeConnectionFailed, nil),
			want: true,
		},
		{
			name: "non-retryable invalid config error",
			err:  NewTransportError("Connect", ErrCodeInvalidConfig, nil),
			want: false,
		},
		{
			name: "non-retryable not connected error",
			err:  NewTransportError("SendAudio", ErrCodeNotConnected, nil),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non-Transport error",
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
