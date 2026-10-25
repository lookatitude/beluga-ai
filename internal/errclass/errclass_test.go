package errclass

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/internal/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeForStatus(t *testing.T) {
	tests := []struct {
		status    int
		want      core.ErrorCode
		retryable bool
	}{
		{400, core.ErrInvalidInput, false},
		{401, core.ErrAuth, false},
		{402, core.ErrBudgetExhausted, false},
		{403, core.ErrAuth, false},
		{404, core.ErrNotFound, false},
		{408, core.ErrTimeout, true},
		{413, core.ErrInvalidInput, false},
		{422, core.ErrInvalidInput, false},
		{429, core.ErrRateLimit, true},
		{500, core.ErrProviderDown, true},
		{502, core.ErrProviderDown, true},
		{503, core.ErrProviderDown, true},
		{529, core.ErrProviderDown, true},
		{302, core.ErrProviderDown, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.want, CodeForStatus(tt.status))
			err := Classify("p: op", tt.status, errors.New("boom"))
			assert.Equal(t, tt.retryable, core.IsRetryable(err))
		})
	}
}

type awsLike struct{ code int }

func (e *awsLike) Error() string       { return "aws error" }
func (e *awsLike) HTTPStatusCode() int { return e.code }

func TestStatusCode(t *testing.T) {
	assert.Equal(t, 0, StatusCode(errors.New("x")))
	assert.Equal(t, 429, StatusCode(fmt.Errorf("w: %w", &httpclient.APIError{StatusCode: 429})))
	assert.Equal(t, 403, StatusCode(fmt.Errorf("op: %w", &awsLike{code: 403})))
}

func TestClassify(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.NoError(t, Classify("p", 0, nil))
	})

	t.Run("status from chain", func(t *testing.T) {
		err := Classify("openai: generate failed", 0, &httpclient.APIError{StatusCode: 401})
		var ce *core.Error
		require.ErrorAs(t, err, &ce)
		assert.Equal(t, core.ErrAuth, ce.Code)
		assert.False(t, core.IsRetryable(err))
		assert.Contains(t, err.Error(), "openai: generate failed")
	})

	t.Run("existing core.Error keeps code", func(t *testing.T) {
		inner := core.Errorf(core.ErrGuardBlocked, "blocked")
		err := Classify("p", 503, inner)
		var ce *core.Error
		require.ErrorAs(t, err, &ce)
		assert.Equal(t, core.ErrGuardBlocked, ce.Code)
		assert.False(t, core.IsRetryable(err))
	})

	t.Run("canceled is not retryable", func(t *testing.T) {
		err := Classify("p", 0, fmt.Errorf("x: %w", context.Canceled))
		assert.ErrorIs(t, err, context.Canceled)
		assert.False(t, core.IsRetryable(err))
	})

	t.Run("deadline is timeout", func(t *testing.T) {
		err := Classify("p", 0, context.DeadlineExceeded)
		assert.True(t, errors.Is(err, core.NewError("", core.ErrTimeout, "", nil)))
		assert.True(t, core.IsRetryable(err))
	})

	t.Run("network error is provider down", func(t *testing.T) {
		err := Classify("p", 0, &url.Error{Op: "Post", URL: "http://h", Err: &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}})
		assert.True(t, errors.Is(err, core.NewError("", core.ErrProviderDown, "", nil)))
		assert.True(t, core.IsRetryable(err))
	})

	t.Run("unexpected EOF is provider down", func(t *testing.T) {
		assert.True(t, core.IsRetryable(Classify("p", 0, io.ErrUnexpectedEOF)))
	})

	t.Run("unknown error is not retryable", func(t *testing.T) {
		err := Classify("p", 0, errors.New("decode: invalid character"))
		assert.False(t, core.IsRetryable(err))
		assert.Contains(t, err.Error(), "decode")
	})

	t.Run("redacts credentials", func(t *testing.T) {
		err := Classify("google", 0, &url.Error{Op: "Post", URL: "http://h/x?key=SECRET", Err: errors.New("refused")})
		assert.NotContains(t, err.Error(), "SECRET")
	})

	t.Run("explicit status wins over chain", func(t *testing.T) {
		err := Classify("p", http.StatusTooManyRequests, errors.New("throttled"))
		assert.True(t, errors.Is(err, core.NewError("", core.ErrRateLimit, "", nil)))
	})
}

func TestWrap(t *testing.T) {
	inner := core.Errorf(core.ErrBudgetExhausted, "over budget")
	e := Wrap(core.ErrInvalidInput, "step %q: %w", "s1", inner)
	assert.Equal(t, core.ErrBudgetExhausted, e.Code)
	assert.ErrorIs(t, e, inner)

	plain := Wrap(core.ErrInvalidInput, "step: %w", errors.New("x"))
	assert.Equal(t, core.ErrInvalidInput, plain.Code)

	noWrap := Wrap(core.ErrInvalidInput, "no cause")
	assert.Equal(t, core.ErrInvalidInput, noWrap.Code)
}
