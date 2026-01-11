package mock

import (
	"bytes"
	"io"
	"net/http"
	"sync"
)

// HTTPClient is an interface for HTTP client operations.
// This allows us to mock HTTP requests in tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// MockHTTPClient is a mock implementation of HTTPClient for testing.
type MockHTTPClient struct {
	mu            sync.RWMutex
	responses    map[string]*MockResponse
	defaultResp  *MockResponse
	requestCount map[string]int
}

// MockResponse represents a mock HTTP response.
type MockResponse struct {
	StatusCode int
	Body       []byte
	Header     http.Header
	Error      error
}

// NewMockHTTPClient creates a new mock HTTP client.
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses:    make(map[string]*MockResponse),
		requestCount: make(map[string]int),
	}
}

// SetResponse sets a mock response for a specific URL pattern.
func (m *MockHTTPClient) SetResponse(urlPattern string, resp *MockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[urlPattern] = resp
}

// SetDefaultResponse sets the default response for unmatched URLs.
func (m *MockHTTPClient) SetDefaultResponse(resp *MockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultResp = resp
}

// GetRequestCount returns the number of times a URL was requested.
func (m *MockHTTPClient) GetRequestCount(urlPattern string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestCount[urlPattern]
}

// Do implements HTTPClient interface.
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	m.requestCount[req.URL.String()]++
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find matching response
	var resp *MockResponse
	for pattern, r := range m.responses {
		if req.URL.String() == pattern || contains(req.URL.String(), pattern) {
			resp = r
			break
		}
	}

	if resp == nil {
		resp = m.defaultResp
	}

	if resp == nil {
		// Default error response
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       http.NoBody,
			Header:     make(http.Header),
		}, nil
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	// Create response body
	var body io.ReadCloser
	if resp.Body != nil {
		body = io.NopCloser(io.Reader(bytes.NewReader(resp.Body)))
	} else {
		body = http.NoBody
	}

	response := &http.Response{
		StatusCode: resp.StatusCode,
		Body:       body,
		Header:      resp.Header,
	}

	if response.Header == nil {
		response.Header = make(http.Header)
	}

	return response, nil
}

// contains checks if a string contains a pattern (simple substring match).
func contains(s, pattern string) bool {
	return len(pattern) > 0 && len(s) >= len(pattern) && s[:len(pattern)] == pattern
}
