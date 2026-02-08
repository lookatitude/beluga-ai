// Package httpclient provides a shared HTTP client with retry, SSE streaming,
// and typed JSON helpers used by providers without dedicated Go SDKs.
package httpclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"math"
	"math/rand/v2"
	"net/http"
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
// It retries on 429/503 with exponential backoff.
func DoJSON[T any](ctx context.Context, c *Client, method, path string, body any) (T, error) {
	var zero T

	for attempt := range c.retries + 1 {
		resp, err := c.Do(ctx, method, path, body, nil)
		if err != nil {
			return zero, err
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			defer resp.Body.Close()
			var result T
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return zero, fmt.Errorf("httpclient: decode response: %w", err)
			}
			return result, nil
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if isRetryable(resp.StatusCode) && attempt < c.retries {
			delay := retryDelay(resp, c.backoff, attempt)
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
		// Try to extract a message from JSON error body.
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
		return zero, apiErr
	}

	return zero, fmt.Errorf("httpclient: exhausted retries")
}

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

// StreamSSE opens an SSE connection and returns an iterator of events.
func StreamSSE(ctx context.Context, c *Client, path string) iter.Seq2[SSEEvent, error] {
	return func(yield func(SSEEvent, error) bool) {
		resp, err := c.Do(ctx, http.MethodGet, path, nil, map[string]string{
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

		scanner := bufio.NewScanner(resp.Body)
		var event SSEEvent

		for scanner.Scan() {
			if ctx.Err() != nil {
				yield(SSEEvent{}, ctx.Err())
				return
			}

			line := scanner.Text()

			if line == "" {
				// Empty line = dispatch event.
				if event.Data != "" || event.Event != "" {
					if !yield(event, nil) {
						return
					}
					event = SSEEvent{}
				}
				continue
			}

			// Ignore comments.
			if strings.HasPrefix(line, ":") {
				continue
			}

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

		if err := scanner.Err(); err != nil {
			yield(SSEEvent{}, fmt.Errorf("httpclient: sse scan: %w", err))
			return
		}

		// Dispatch final event if not terminated by empty line.
		if event.Data != "" || event.Event != "" {
			yield(event, nil)
		}
	}
}
