package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"no query", "https://api.example.com/v1/chat", "https://api.example.com/v1/chat"},
		{"google key", "https://g.example/models/x:embed?key=SECRET-KEY-123", "https://g.example/models/x:embed?key=REDACTED"},
		{"api_key keeps others", "https://x/y?a=1&api_key=s3cr3t&b=2", "https://x/y?a=1&api_key=REDACTED&b=2"},
		{"token and access_token", "https://x/?token=t1&access_token=t2", "https://x/?token=REDACTED&access_token=REDACTED"},
		{"case insensitive", "https://x/?API_KEY=abc&X-Goog-Api-Key=def", "https://x/?API_KEY=REDACTED&X-Goog-Api-Key=REDACTED"},
		{"signature", "https://s3/x?X-Amz-Signature=abc&X-Amz-Date=1", "https://s3/x?X-Amz-Signature=REDACTED&X-Amz-Date=1"},
		{"non-sensitive untouched", "https://x/?model=gpt&limit=10", "https://x/?model=gpt&limit=10"},
		{"empty value untouched", "https://x/?key=", "https://x/?key="},
		{"userinfo password", "https://user:hunter2@db.example/x", "https://user:REDACTED@db.example/x"},
		{"embedded in message", `Post "http://127.0.0.1:1/m:batch?key=SECRET": dial tcp: refused`, `Post "http://127.0.0.1:1/m:batch?key=REDACTED": dial tcp: refused`},
		{"wss url", "wss://live.example/ws?key=abc", "wss://live.example/ws?key=REDACTED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RedactURL(tt.in))
		})
	}
}

func TestRedactError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.NoError(t, RedactError(nil))
	})

	t.Run("clean error returned unchanged", func(t *testing.T) {
		e := errors.New("plain")
		assert.Same(t, e, RedactError(e))
	})

	t.Run("url.Error redacted in place and chain preserved", func(t *testing.T) {
		inner := errors.New("connection refused")
		ue := &url.Error{Op: "Post", URL: "http://h/x?key=SECRET", Err: inner}
		wrapped := fmt.Errorf("provider: %w", ue)
		got := RedactError(wrapped)
		assert.NotContains(t, got.Error(), "SECRET")
		assert.ErrorIs(t, got, inner)
		var asURL *url.Error
		require.ErrorAs(t, got, &asURL)
		assert.NotContains(t, asURL.URL, "SECRET")
	})

	t.Run("plain message with key is wrapped", func(t *testing.T) {
		sentinel := errors.New("sentinel")
		e := fmt.Errorf("GET https://h/?api_key=SECRET failed: %w", sentinel)
		got := RedactError(e)
		assert.NotContains(t, got.Error(), "SECRET")
		assert.ErrorIs(t, got, sentinel)
	})

	t.Run("joined errors", func(t *testing.T) {
		ue := &url.Error{Op: "Get", URL: "http://h/?token=SECRET", Err: errors.New("x")}
		got := RedactError(errors.Join(errors.New("a"), ue))
		assert.NotContains(t, got.Error(), "SECRET")
	})
}

// TestDo_RedactsKeyInNetworkError is the FI-05 reproduction: an API key in the
// query string must never appear in the error returned for a network failure.
func TestDo_RedactsKeyInNetworkError(t *testing.T) {
	c := New(WithTimeout(2 * time.Second))
	resp, err := c.Do(context.Background(), http.MethodPost, "http://127.0.0.1:1/models/x:batchEmbedContents?key=SECRET-KEY-123", map[string]string{"a": "b"}, nil)
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "SECRET-KEY-123")
	assert.Contains(t, err.Error(), "key=REDACTED")

	_, err = DoJSON[map[string]any](context.Background(), c, http.MethodGet, "http://127.0.0.1:1/x?key=SECRET-KEY-123", nil)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "SECRET-KEY-123")

	for _, e := range StreamSSE(context.Background(), c, "http://127.0.0.1:1/s?key=SECRET-KEY-123") {
		require.Error(t, e)
		assert.False(t, strings.Contains(e.Error(), "SECRET-KEY-123"))
	}
}

func TestDo_RedactsKeyInBadURL(t *testing.T) {
	c := New()
	_, err := c.Do(context.Background(), "BAD METHOD", "http://h/x?key=SECRET", nil, nil) //nolint:bodyclose // error path, no body
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "SECRET")
}

func TestDialWS_RedactsKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := DialWS(ctx, "ws://127.0.0.1:1/live?key=SECRET-KEY-123", nil)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "SECRET-KEY-123")
}
