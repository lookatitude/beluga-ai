package httpclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client wraps net/http.Client with retry, context propagation, and typed helpers.
type Client struct {
	http    *http.Client
	baseURL string
	headers map[string]string
	retries int
	backoff time.Duration
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets the base URL prepended to all request paths.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(url, "/")
	}
}

// WithHeader adds a default header sent with every request.
func WithHeader(key, value string) Option {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.http.Timeout = d
	}
}

// WithRetries sets the maximum number of retry attempts for retryable errors.
func WithRetries(n int) Option {
	return func(c *Client) {
		c.retries = n
	}
}

// WithBackoff sets the base duration for exponential backoff between retries.
func WithBackoff(d time.Duration) Option {
	return func(c *Client) {
		c.backoff = d
	}
}

// WithBearerToken sets the Authorization header to "Bearer <token>".
func WithBearerToken(token string) Option {
	return func(c *Client) {
		c.headers["Authorization"] = "Bearer " + token
	}
}

// New creates a Client with the given options.
func New(opts ...Option) *Client {
	c := &Client{
		http:    &http.Client{Timeout: 30 * time.Second},
		headers: make(map[string]string),
		retries: 0,
		backoff: 500 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// APIError represents an HTTP error response from an API.
type APIError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("api error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("api error (status %d): %s", e.StatusCode, e.Body)
}

// Do sends an HTTP request and returns the raw response.
// The caller is responsible for closing the response body.
func (c *Client) Do(ctx context.Context, method, path string, body any, headers map[string]string) (*http.Response, error) {
	url := path
	if c.baseURL != "" && !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		url = c.baseURL + "/" + strings.TrimLeft(path, "/")
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("httpclient: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("httpclient: create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.http.Do(req)
}

// isRetryable returns true if the status code warrants a retry.
func isRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable
}

// isNetworkError returns true if the error is a temporary or timeout network error.
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Unwrap url.Error if present (HTTP client wraps errors in url.Error).
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}

	// Check for net.Error interface (Temporary/Timeout).
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Temporary() || netErr.Timeout()
	}

	// Check for net.OpError (connection refused, DNS failures, etc.).
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Check for EOF and unexpected EOF (connection closed unexpectedly).
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// Check for "connection reset by peer" and similar errors.
	errStr := err.Error()
	if strings.Contains(errStr, "connection reset") ||
	   strings.Contains(errStr, "broken pipe") ||
	   strings.Contains(errStr, "connection refused") {
		return true
	}

	return false
}

// retryDelay computes the delay for a retry attempt, respecting Retry-After headers.
func retryDelay(resp *http.Response, baseBackoff time.Duration, attempt int) time.Duration {
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if seconds, err := strconv.Atoi(ra); err == nil {
				return time.Duration(seconds) * time.Second
			}
		}
	}
	// Exponential backoff with jitter.
	exp := math.Pow(2, float64(attempt))
	base := time.Duration(float64(baseBackoff) * exp)
	jitter := time.Duration(rand.Int64N(int64(base)/2 + 1))
	return base + jitter
}

// DoJSON sends an HTTP request and decodes the JSON response into T.
// It retries on 429/503 and network errors with exponential backoff.
func DoJSON[T any](ctx context.Context, c *Client, method, path string, body any) (T, error) {
	var zero T

	for attempt := range c.retries + 1 {
		result, err, done := doJSONAttempt[T](ctx, c, method, path, body, attempt)
		if done {
			return result, err
		}
	}

	return zero, fmt.Errorf("httpclient: exhausted retries")
}

// doJSONAttempt executes a single attempt of DoJSON. It returns (result, err, done).
// done=false means the caller should continue to the next retry attempt.
func doJSONAttempt[T any](ctx context.Context, c *Client, method, path string, body any, attempt int) (T, error, bool) {
	var zero T

	resp, err := c.Do(ctx, method, path, body, nil)
	if err != nil {
		return zero, err, !shouldRetryNetErr(ctx, c, err, attempt)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return decodeJSONResponse[T](resp)
	}

	apiErr, shouldRetry := handleErrorResponse(ctx, c, resp, attempt)
	return zero, apiErr, !shouldRetry
}

// shouldRetryNetErr checks if a network error should be retried, and waits if so.
// Returns true if the caller should retry (continue to next attempt).
func shouldRetryNetErr(ctx context.Context, c *Client, err error, attempt int) bool {
	if !isNetworkError(err) || attempt >= c.retries {
		return false
	}
	return waitForRetry(ctx, nil, c.backoff, attempt)
}

// decodeJSONResponse reads and decodes a successful JSON response.
func decodeJSONResponse[T any](resp *http.Response) (T, error, bool) {
	defer resp.Body.Close()
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("httpclient: decode response: %w", err), true
	}
	return result, nil, true
}

// handleErrorResponse processes a non-2xx response, retrying if appropriate.
// Returns the error and whether the caller should retry.
func handleErrorResponse(ctx context.Context, c *Client, resp *http.Response, attempt int) (error, bool) {
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if isRetryable(resp.StatusCode) && attempt < c.retries {
		if waitForRetry(ctx, resp, c.backoff, attempt) {
			return newAPIError(resp.StatusCode, respBody), true
		}
		// Context was cancelled during retry wait.
		if err := ctx.Err(); err != nil {
			return err, false
		}
	}

	return newAPIError(resp.StatusCode, respBody), false
}

// waitForRetry waits for the retry delay, respecting context cancellation.
// Returns true if the caller should continue to the next attempt, or false
// if the context was cancelled.
func waitForRetry(ctx context.Context, resp *http.Response, backoff time.Duration, attempt int) bool {
	delay := retryDelay(resp, backoff, attempt)
	select {
	case <-ctx.Done():
		return false
	case <-time.After(delay):
		return true
	}
}

// newAPIError creates an APIError from a response status code and body,
// attempting to extract an error message from JSON.
func newAPIError(statusCode int, respBody []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Body:       string(respBody),
	}
	var errBody struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(respBody, &errBody) == nil {
		if errBody.Error.Message != "" {
			apiErr.Message = errBody.Error.Message
		} else if errBody.Message != "" {
			apiErr.Message = errBody.Message
		}
	}
	return apiErr
}

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

// StreamSSE opens an SSE connection using GET and returns an iterator of events.
func StreamSSE(ctx context.Context, c *Client, path string) iter.Seq2[SSEEvent, error] {
	return StreamSSEWithBody(ctx, c, http.MethodGet, path, nil)
}

// StreamSSEWithBody opens an SSE connection with the specified method and optional body.
// Many LLM providers require POST for SSE streaming.
func StreamSSEWithBody(ctx context.Context, c *Client, method, path string, body any) iter.Seq2[SSEEvent, error] {
	return func(yield func(SSEEvent, error) bool) {
		resp, err := c.Do(ctx, method, path, body, map[string]string{
			"Accept":        "text/event-stream",
			"Cache-Control": "no-cache",
		})
		if err != nil {
			yield(SSEEvent{}, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			yield(SSEEvent{}, &APIError{
				StatusCode: resp.StatusCode,
				Body:       string(body),
			})
			return
		}

		scanSSEStream(ctx, resp.Body, yield)
	}
}

// scanSSEStream reads SSE events from a reader and yields them.
func scanSSEStream(ctx context.Context, r io.Reader, yield func(SSEEvent, error) bool) {
	scanner := bufio.NewScanner(r)
	var event SSEEvent

	for scanner.Scan() {
		if ctx.Err() != nil {
			yield(SSEEvent{}, ctx.Err())
			return
		}

		line := scanner.Text()

		if line == "" {
			if !dispatchSSEEvent(&event, yield) {
				return
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		parseSSEField(line, &event)
	}

	if err := scanner.Err(); err != nil {
		yield(SSEEvent{}, fmt.Errorf("httpclient: sse scan: %w", err))
		return
	}

	// Dispatch final event if not terminated by empty line.
	dispatchSSEEvent(&event, yield)
}

// dispatchSSEEvent yields the event if it has data, then resets it.
// Returns false if the yield function returns false (consumer stopped).
func dispatchSSEEvent(event *SSEEvent, yield func(SSEEvent, error) bool) bool {
	if event.Data == "" && event.Event == "" {
		return true
	}
	ok := yield(*event, nil)
	*event = SSEEvent{}
	return ok
}

// parseSSEField parses a single SSE field line and updates the event.
func parseSSEField(line string, event *SSEEvent) {
	field, value, _ := strings.Cut(line, ":")
	value = strings.TrimPrefix(value, " ")

	switch field {
	case "event":
		event.Event = value
	case "data":
		if event.Data != "" {
			event.Data += "\n"
		}
		event.Data += value
	case "id":
		event.ID = value
	case "retry":
		if ms, err := strconv.Atoi(value); err == nil {
			event.Retry = ms
		}
	}
}
