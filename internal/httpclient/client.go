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
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	// maxRetryAfter caps how long a server-supplied Retry-After header can
	// make the client sleep between attempts.
	maxRetryAfter = 60 * time.Second

	// maxErrorBodyBytes bounds how much of a non-2xx response body is read
	// into an APIError.
	maxErrorBodyBytes = 1 << 20 // 1 MiB

	// maxResponseBytes bounds how much of a successful JSON response body is
	// decoded by DoJSON.
	maxResponseBytes = 64 << 20 // 64 MiB
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
		return nil, RedactError(fmt.Errorf("httpclient: create request: %w", err))
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

	resp, err := c.http.Do(req)
	if err != nil {
		// *url.Error renders the full request URL, which may carry an API key
		// in its query string (e.g. Google's ?key=). Scrub it before the error
		// can reach logs, spans or HTTP responses.
		return nil, RedactError(err)
	}
	return resp, nil
}

// isIdempotent reports whether an HTTP method can be safely re-sent after the
// request may already have reached the server (RFC 9110 section 9.2.2).
func isIdempotent(method string) bool {
	switch strings.ToUpper(method) {
	case "", http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace,
		http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}

// isRetryable returns true if the status code warrants a retry for method.
// 429 and 503 mean the server did not process the request, so they are
// retried for every method. 502 and 504 come from a gateway that may have
// forwarded the request, so they are retried only for idempotent methods.
func isRetryable(method string, statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return true
	case http.StatusBadGateway, http.StatusGatewayTimeout:
		return isIdempotent(method)
	}
	return false
}

// transientErrnos are socket errors that indicate a transient network fault.
var transientErrnos = []syscall.Errno{
	syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ECONNABORTED,
	syscall.EPIPE, syscall.ETIMEDOUT, syscall.EHOSTUNREACH, syscall.ENETUNREACH,
	syscall.ENETDOWN, syscall.EAGAIN,
}

// IsNetworkError reports whether err is a transient transport-level failure
// (timeout, refused or reset connection, DNS hiccup, connection closed
// mid-response). Context cancellation is never a network error.
//
// It deliberately does not use the deprecated net.Error.Temporary, whose
// answer is ill-defined (and false for ECONNREFUSED); transient conditions are
// classified explicitly instead.
func IsNetworkError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	for _, errno := range transientErrnos {
		if errors.Is(err, errno) {
			return true
		}
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.IsTimeout || dnsErr.IsTemporary
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// Fallback for errors that lost their typed cause.
	errStr := err.Error()
	return strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection refused")
}

// isNetworkError is the internal alias kept for existing call sites.
func isNetworkError(err error) bool { return IsNetworkError(err) }

// requestNotSent reports whether err happened before any byte of the request
// could have reached the server (dial or DNS failure, refused connection).
func requestNotSent(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "dial"
}

// shouldRetryNetwork reports whether a transport error is safe to retry for
// method. Idempotent requests retry on any transient network error; other
// methods (POST, PATCH) only when the request provably never left the client,
// so a write is never duplicated.
func shouldRetryNetwork(method string, err error) bool {
	if !isNetworkError(err) {
		return false
	}
	return isIdempotent(method) || requestNotSent(err)
}

// parseRetryAfter parses a Retry-After header in either delta-seconds or
// HTTP-date form. It returns false when the header is absent or invalid.
func parseRetryAfter(ra string, now time.Time) (time.Duration, bool) {
	ra = strings.TrimSpace(ra)
	if ra == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(ra); err == nil {
		if seconds < 0 {
			return 0, false
		}
		if seconds > int(maxRetryAfter/time.Second) {
			return maxRetryAfter, true
		}
		return time.Duration(seconds) * time.Second, true
	}
	if t, err := http.ParseTime(ra); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// retryDelay computes the delay for a retry attempt, respecting Retry-After
// headers (capped at maxRetryAfter).
func retryDelay(resp *http.Response, baseBackoff time.Duration, attempt int) time.Duration {
	if resp != nil {
		if d, ok := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); ok {
			return min(d, maxRetryAfter)
		}
	}
	// Exponential backoff with jitter.
	exp := math.Pow(2, float64(attempt))
	base := time.Duration(float64(baseBackoff) * exp)
	jitter := time.Duration(rand.Int64N(int64(base)/2 + 1)) // #nosec G404 -- HTTP retry jitter is not security-sensitive
	return base + jitter
}

// DoJSON sends an HTTP request and decodes the JSON response into T.
// It retries on 429/503 (and 502/504 for idempotent methods) and on transient
// network errors with exponential backoff. Network errors are retried for
// non-idempotent methods (POST, PATCH) only when the request was never sent,
// so writes are not duplicated. Server Retry-After values are capped at 60s.
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
		return zero, err, !shouldRetryNetErr(ctx, c, method, err, attempt)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return decodeJSONResponse[T](resp)
	}

	apiErr, shouldRetry := handleErrorResponse(ctx, c, method, resp, attempt)
	return zero, apiErr, !shouldRetry
}

// shouldRetryNetErr checks if a network error should be retried, and waits if so.
// Returns true if the caller should retry (continue to next attempt).
func shouldRetryNetErr(ctx context.Context, c *Client, method string, err error, attempt int) bool {
	if attempt >= c.retries || !shouldRetryNetwork(method, err) {
		return false
	}
	return waitForRetry(ctx, nil, c.backoff, attempt)
}

// decodeJSONResponse reads and decodes a successful JSON response.
func decodeJSONResponse[T any](resp *http.Response) (T, error, bool) {
	defer resp.Body.Close()
	var result T
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&result); err != nil {
		return result, fmt.Errorf("httpclient: decode response: %w", err), true
	}
	return result, nil, true
}

// handleErrorResponse processes a non-2xx response, retrying if appropriate.
// Returns the error and whether the caller should retry.
func handleErrorResponse(ctx context.Context, c *Client, method string, resp *http.Response, attempt int) (error, bool) {
	respBody := readErrorBody(resp.Body)
	_ = resp.Body.Close() // #nosec G104 -- close error in error-handling path is not actionable

	if isRetryable(method, resp.StatusCode) && attempt < c.retries {
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

// readErrorBody reads at most maxErrorBodyBytes of an error response body.
func readErrorBody(r io.Reader) []byte {
	b, _ := io.ReadAll(io.LimitReader(r, maxErrorBodyBytes))
	return b
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
			yield(SSEEvent{}, newAPIError(resp.StatusCode, readErrorBody(resp.Body)))
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
