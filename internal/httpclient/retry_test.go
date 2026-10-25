package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		in     string
		want   time.Duration
		wantOK bool
	}{
		{"empty", "", 0, false},
		{"seconds", "5", 5 * time.Second, true},
		{"zero", "0", 0, true},
		{"negative", "-3", 0, false},
		{"huge capped", "86400", maxRetryAfter, true},
		{"overflow capped", "999999999999", maxRetryAfter, true},
		{"http date", now.Add(10 * time.Second).Format(http.TimeFormat), 10 * time.Second, true},
		{"past http date", now.Add(-time.Hour).Format(http.TimeFormat), 0, true},
		{"garbage", "soon", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseRetryAfter(tt.in, now)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRetryDelay_CapsRetryAfter(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "3600")
	assert.Equal(t, maxRetryAfter, retryDelay(resp, time.Millisecond, 0))

	resp.Header.Set("Retry-After", time.Now().Add(2*time.Hour).UTC().Format(http.TimeFormat))
	assert.Equal(t, maxRetryAfter, retryDelay(resp, time.Millisecond, 0))
}

func TestIsRetryableStatus(t *testing.T) {
	tests := []struct {
		method string
		status int
		want   bool
	}{
		{http.MethodPost, http.StatusTooManyRequests, true},
		{http.MethodPost, http.StatusServiceUnavailable, true},
		{http.MethodPost, http.StatusBadGateway, false},
		{http.MethodPost, http.StatusGatewayTimeout, false},
		{http.MethodGet, http.StatusBadGateway, true},
		{http.MethodGet, http.StatusGatewayTimeout, true},
		{http.MethodGet, http.StatusInternalServerError, false},
		{http.MethodGet, http.StatusBadRequest, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, isRetryable(tt.method, tt.status), "%s %d", tt.method, tt.status)
	}
}

func TestShouldRetryNetwork(t *testing.T) {
	dialRefused := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	readReset := &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}
	tests := []struct {
		name   string
		method string
		err    error
		want   bool
	}{
		{"GET plain error", http.MethodGet, errors.New("x"), false},
		{"GET reset", http.MethodGet, readReset, true},
		{"POST reset not retried", http.MethodPost, readReset, false},
		{"POST EOF not retried", http.MethodPost, wrapEOF(), false},
		{"POST dial refused retried", http.MethodPost, dialRefused, true},
		{"POST DNS retried", http.MethodPost, &net.DNSError{Err: "tmp", IsTemporary: true}, true},
		{"PUT reset retried", http.MethodPut, readReset, true},
		{"canceled", http.MethodGet, context.Canceled, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldRetryNetwork(tt.method, tt.err))
		})
	}
}

func wrapEOF() error { return fmt.Errorf("read body: %w", io.ErrUnexpectedEOF) }

// hijackCloseServer closes the first n connections without a response.
func hijackCloseServer(t *testing.T, n int32, attempts *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) <= n {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("hijacking not supported")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_ = conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"ok"}`))
	}))
}

func TestDoJSON_PostNotRetriedAfterRequestSent(t *testing.T) {
	var attempts atomic.Int32
	srv := hijackCloseServer(t, 1, &attempts)
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithRetries(3), WithBackoff(time.Millisecond))
	_, err := DoJSON[testResponse](context.Background(), c, http.MethodPost, "/data", map[string]string{"a": "b"})
	require.Error(t, err)
	assert.Equal(t, int32(1), attempts.Load(), "a POST that reached the server must not be re-sent")
}

func TestDoJSON_PostRetriedOnDialFailure(t *testing.T) {
	// Grab a free port, then close it so the first dial is refused.
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	c := New(WithRetries(2), WithBackoff(time.Millisecond))
	start := time.Now()
	_, err = DoJSON[testResponse](context.Background(), c, http.MethodPost, "http://"+addr+"/x", map[string]string{"a": "b"})
	require.Error(t, err)
	assert.True(t, IsNetworkError(err), "dial refused should be classified as a network error: %v", err)
	assert.Less(t, time.Since(start), 5*time.Second)
}

func TestDoJSON_502RetriedForGetNotPost(t *testing.T) {
	for _, tc := range []struct {
		method string
		want   int32
	}{{http.MethodGet, 2}, {http.MethodPost, 1}} {
		t.Run(tc.method, func(t *testing.T) {
			var attempts atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if attempts.Add(1) == 1 {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				_, _ = w.Write([]byte(`{"name":"ok"}`))
			}))
			defer srv.Close()
			c := New(WithBaseURL(srv.URL), WithRetries(2), WithBackoff(time.Millisecond))
			_, _ = DoJSON[testResponse](context.Background(), c, tc.method, "/x", nil)
			assert.Equal(t, tc.want, attempts.Load())
		})
	}
}

func TestDoJSON_ErrorBodyIsBounded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(strings.Repeat("a", maxErrorBodyBytes+4096)))
	}))
	defer srv.Close()
	c := New(WithBaseURL(srv.URL))
	_, err := DoJSON[testResponse](context.Background(), c, http.MethodGet, "/x", nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, maxErrorBodyBytes, len(apiErr.Body))
}

func TestStreamSSE_ErrorStatusParsesMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer srv.Close()
	c := New(WithBaseURL(srv.URL))
	for _, err := range StreamSSE(context.Background(), c, "/s") {
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
		assert.Equal(t, "bad key", apiErr.Message)
	}
}
