package httpclient

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_Defaults(t *testing.T) {
	c := New()
	assert.NotNil(t, c.http)
	assert.Equal(t, 30*time.Second, c.http.Timeout)
	assert.Empty(t, c.baseURL)
	assert.Empty(t, c.headers)
	assert.Equal(t, 0, c.retries)
	assert.Equal(t, 500*time.Millisecond, c.backoff)
}

func TestNew_WithOptions(t *testing.T) {
	c := New(
		WithBaseURL("https://api.example.com"),
		WithHeader("X-Custom", "value"),
		WithTimeout(10*time.Second),
		WithRetries(3),
		WithBackoff(1*time.Second),
		WithBearerToken("tok123"),
	)
	assert.Equal(t, "https://api.example.com", c.baseURL)
	assert.Equal(t, "value", c.headers["X-Custom"])
	assert.Equal(t, 10*time.Second, c.http.Timeout)
	assert.Equal(t, 3, c.retries)
	assert.Equal(t, 1*time.Second, c.backoff)
	assert.Equal(t, "Bearer tok123", c.headers["Authorization"])
}

type testResponse struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type testRequest struct {
	Input string `json:"input"`
}

func TestDoJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req testRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "hello", req.Input)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testResponse{Name: "result", Value: 42})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	resp, err := DoJSON[testResponse](context.Background(), c, http.MethodPost, "/test", testRequest{Input: "hello"})
	require.NoError(t, err)
	assert.Equal(t, "result", resp.Name)
	assert.Equal(t, 42, resp.Value)
}

func TestDoJSON_Retry429(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testResponse{Name: "ok", Value: 1})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithRetries(3), WithBackoff(1*time.Millisecond))
	resp, err := DoJSON[testResponse](context.Background(), c, http.MethodGet, "/data", nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Name)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestDoJSON_Retry503(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("unavailable"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testResponse{Name: "recovered", Value: 2})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithRetries(2), WithBackoff(1*time.Millisecond))
	resp, err := DoJSON[testResponse](context.Background(), c, http.MethodGet, "/data", nil)
	require.NoError(t, err)
	assert.Equal(t, "recovered", resp.Name)
	assert.Equal(t, int32(2), attempts.Load())
}

func TestDoJSON_NoRetryOn400(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"bad request"}`))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithRetries(3), WithBackoff(1*time.Millisecond))
	_, err := DoJSON[testResponse](context.Background(), c, http.MethodGet, "/data", nil)
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.StatusCode)
	assert.Equal(t, "bad request", apiErr.Message)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestDoJSON_MaxRetries(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithRetries(2), WithBackoff(1*time.Millisecond))
	_, err := DoJSON[testResponse](context.Background(), c, http.MethodGet, "/data", nil)
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 429, apiErr.StatusCode)
	// 1 initial + 2 retries = 3 attempts
	assert.Equal(t, int32(3), attempts.Load())
}

func TestDoJSON_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	c := New(WithBaseURL(srv.URL), WithRetries(10), WithBackoff(5*time.Second))

	done := make(chan error, 1)
	go func() {
		_, err := DoJSON[testResponse](ctx, c, http.MethodGet, "/data", nil)
		done <- err
	}()

	// Cancel shortly after the first attempt.
	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-done
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDo_Headers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "default-val", r.Header.Get("X-Default"))
		assert.Equal(t, "per-req-val", r.Header.Get("X-PerReq"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithHeader("X-Default", "default-val"))
	resp, err := c.Do(context.Background(), http.MethodGet, "/test", nil, map[string]string{
		"X-PerReq": "per-req-val",
	})
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDo_BearerToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithBearerToken("mytoken"))
	resp, err := c.Do(context.Background(), http.MethodGet, "/auth", nil, nil)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAPIError(t *testing.T) {
	err := &APIError{StatusCode: 404, Body: `{"error":"not found"}`, Message: "not found"}
	assert.Equal(t, "api error (status 404): not found", err.Error())

	err2 := &APIError{StatusCode: 500, Body: "internal error"}
	assert.Equal(t, "api error (status 500): internal error", err2.Error())
}

func TestStreamSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))
		assert.Equal(t, "no-cache", r.Header.Get("Cache-Control"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		// Event 1: named event with data.
		fmt.Fprintln(w, "event: message")
		fmt.Fprintln(w, "data: hello world")
		fmt.Fprintln(w, "id: 1")
		fmt.Fprintln(w)
		flusher.Flush()

		// Event 2: multi-line data.
		fmt.Fprintln(w, "data: line1")
		fmt.Fprintln(w, "data: line2")
		fmt.Fprintln(w)
		flusher.Flush()

		// Event 3: retry field.
		fmt.Fprintln(w, ": this is a comment")
		fmt.Fprintln(w, "event: done")
		fmt.Fprintln(w, "data: finished")
		fmt.Fprintln(w, "retry: 5000")
		fmt.Fprintln(w)
		flusher.Flush()
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	var events []SSEEvent
	for ev, err := range StreamSSE(context.Background(), c, "/stream") {
		require.NoError(t, err)
		events = append(events, ev)
	}

	require.Len(t, events, 3)

	assert.Equal(t, "message", events[0].Event)
	assert.Equal(t, "hello world", events[0].Data)
	assert.Equal(t, "1", events[0].ID)

	assert.Equal(t, "", events[1].Event)
	assert.Equal(t, "line1\nline2", events[1].Data)

	assert.Equal(t, "done", events[2].Event)
	assert.Equal(t, "finished", events[2].Data)
	assert.Equal(t, 5000, events[2].Retry)
}

func TestStreamSSE_ContextCancel(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		close(started)

		// Send events until client disconnects.
		for i := range 1000 {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			fmt.Fprintf(w, "data: event %d\n\n", i)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := New(WithBaseURL(srv.URL))
	var count int
	for _, err := range StreamSSE(ctx, c, "/stream") {
		if err != nil {
			break
		}
		count++
		if count >= 3 {
			cancel()
		}
	}

	// Should have received some events but stopped early.
	assert.GreaterOrEqual(t, count, 3)
	assert.Less(t, count, 1000)
}

func TestStreamSSE_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	for _, err := range StreamSSE(context.Background(), c, "/stream") {
		require.Error(t, err)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, 500, apiErr.StatusCode)
		break
	}
}

func TestDo_FullURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Client with a base URL, but Do with a full URL should use the full URL.
	c := New(WithBaseURL("https://other.example.com"))
	resp, err := c.Do(context.Background(), http.MethodGet, srv.URL+"/test", nil, nil)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDoJSON_RetryAfterHeader(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testResponse{Name: "ok", Value: 1})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithRetries(2), WithBackoff(1*time.Millisecond))
	resp, err := DoJSON[testResponse](context.Background(), c, http.MethodGet, "/data", nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Name)
}

func TestDoJSON_RetryOnNetworkError(t *testing.T) {
	var attempts atomic.Int32

	// Create a server that's stopped after first request, simulating network failure.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			// Simulate network error by hijacking and closing connection abruptly.
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("hijacking not supported")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack failed: %v", err)
			}
			conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testResponse{Name: "recovered", Value: 3})
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithRetries(3), WithBackoff(1*time.Millisecond))
	resp, err := DoJSON[testResponse](context.Background(), c, http.MethodGet, "/data", nil)
	require.NoError(t, err)
	assert.Equal(t, "recovered", resp.Name)
	assert.GreaterOrEqual(t, attempts.Load(), int32(3))
}

func TestDoJSON_NoRetryOnNetworkErrorWhenRetriesDisabled(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		// Simulate network error by hijacking and closing connection.
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("hijacking not supported")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack failed: %v", err)
		}
		conn.Close()
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithRetries(0), WithBackoff(1*time.Millisecond))
	_, err := DoJSON[testResponse](context.Background(), c, http.MethodGet, "/data", nil)
	require.Error(t, err)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"EOF", io.EOF, true},
		{"unexpected EOF", io.ErrUnexpectedEOF, true},
		// net.OpError implements net.Error, so it's caught by the net.Error check.
		// With a non-temporary inner error, Temporary()/Timeout() returns false.
		{"net.OpError non-temp", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("refused")}, false},
		// With a temporary inner error, the net.Error check returns true.
		{"net.OpError with temp", &net.OpError{Op: "dial", Net: "tcp", Err: &testNetError{temporary: true}}, true},
		// url.Error unwrapping then net.Error check.
		{"url.Error wrapping temp OpError", &url.Error{Op: "Get", URL: "http://x", Err: &net.OpError{Op: "dial", Net: "tcp", Err: &testNetError{timeout: true}}}, true},
		{"connection reset string", errors.New("read: connection reset by peer"), true},
		{"broken pipe string", errors.New("write: broken pipe"), true},
		{"connection refused string", errors.New("dial: connection refused"), true},
		{"non-network error", errors.New("something else"), false},
		{"context canceled", context.Canceled, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNetworkError(tt.err)
			assert.Equal(t, tt.want, got, "isNetworkError(%v)", tt.err)
		})
	}
}

func TestIsNetworkError_NetError(t *testing.T) {
	// Test net.Error interface with Temporary/Timeout.
	t.Run("timeout error", func(t *testing.T) {
		err := &testNetError{timeout: true}
		assert.True(t, isNetworkError(err))
	})
	t.Run("temporary error", func(t *testing.T) {
		err := &testNetError{temporary: true}
		assert.True(t, isNetworkError(err))
	})
	t.Run("non-temporary non-timeout net.Error", func(t *testing.T) {
		err := &testNetError{}
		assert.False(t, isNetworkError(err))
	})
	t.Run("url.Error wrapping net.Error", func(t *testing.T) {
		err := &url.Error{Op: "Get", URL: "http://x", Err: &testNetError{timeout: true}}
		assert.True(t, isNetworkError(err))
	})
}

// testNetError implements net.Error for testing.
type testNetError struct {
	timeout   bool
	temporary bool
}

func (e *testNetError) Error() string   { return "test net error" }
func (e *testNetError) Timeout() bool   { return e.timeout }
func (e *testNetError) Temporary() bool { return e.temporary }

func TestDo_MarshalError(t *testing.T) {
	c := New(WithBaseURL("http://localhost"))
	// Channels cannot be JSON marshaled.
	_, err := c.Do(context.Background(), http.MethodPost, "/test", make(chan int), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal body")
}

func TestDoJSON_InvalidJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := DoJSON[testResponse](context.Background(), c, http.MethodGet, "/data", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestStreamSSEWithBody_DoError(t *testing.T) {
	// Client with unreachable base URL to trigger Do() error.
	c := New(WithBaseURL("http://127.0.0.1:1"))
	for _, err := range StreamSSEWithBody(context.Background(), c, http.MethodGet, "/stream", nil) {
		require.Error(t, err)
		break
	}
}

func TestStreamSSEWithBody_ScannerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		// Send one valid event.
		fmt.Fprintln(w, "data: hello")
		fmt.Fprintln(w)
		flusher.Flush()

		// Write an oversized line to trigger scanner buffer error.
		huge := make([]byte, bufio.MaxScanTokenSize+1)
		for i := range huge {
			huge[i] = 'x'
		}
		w.Write(huge)
		flusher.Flush()
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	var events []SSEEvent
	var lastErr error
	for ev, err := range StreamSSEWithBody(context.Background(), c, http.MethodPost, "/stream", nil) {
		if err != nil {
			lastErr = err
			break
		}
		events = append(events, ev)
	}
	assert.Len(t, events, 1)
	require.Error(t, lastErr)
	assert.Contains(t, lastErr.Error(), "sse scan")
}

func TestStreamSSEWithBody_FinalEventNoTrailingNewline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Write an event without trailing empty line — stream ends abruptly.
		fmt.Fprint(w, "event: final\ndata: last\n")
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	var events []SSEEvent
	for ev, err := range StreamSSEWithBody(context.Background(), c, http.MethodGet, "/stream", nil) {
		require.NoError(t, err)
		events = append(events, ev)
	}
	require.Len(t, events, 1)
	assert.Equal(t, "final", events[0].Event)
	assert.Equal(t, "last", events[0].Data)
}

func TestStreamSSEWithBody_ContextCancelDuringScan(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		close(started)

		// Keep sending events slowly.
		for i := range 1000 {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			fmt.Fprintf(w, "data: event-%d\n\n", i)
			flusher.Flush()
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := New(WithBaseURL(srv.URL))
	var count int
	var gotCtxErr bool
	for _, err := range StreamSSEWithBody(ctx, c, http.MethodGet, "/stream", nil) {
		if err != nil {
			if errors.Is(err, context.Canceled) {
				gotCtxErr = true
			}
			break
		}
		count++
		if count >= 2 {
			cancel()
		}
	}
	assert.GreaterOrEqual(t, count, 2)
	assert.True(t, gotCtxErr, "expected context.Canceled error")
}

func TestStreamSSEWithBody_POST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req testRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "test input", req.Input)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		fmt.Fprintln(w, "event: response")
		fmt.Fprintf(w, "data: received: %s\n", req.Input)
		fmt.Fprintln(w)
		flusher.Flush()
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	var events []SSEEvent
	for ev, err := range StreamSSEWithBody(context.Background(), c, http.MethodPost, "/stream", testRequest{Input: "test input"}) {
		require.NoError(t, err)
		events = append(events, ev)
	}

	require.Len(t, events, 1)
	assert.Equal(t, "response", events[0].Event)
	assert.Equal(t, "received: test input", events[0].Data)
}
