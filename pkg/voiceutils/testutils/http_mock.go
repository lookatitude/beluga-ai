package testutils

import (
	"io"
	"net/http"
	"strings"
)

// MockRoundTripper is a mock implementation of http.RoundTripper for testing.
type MockRoundTripper struct {
	// Response is the response to return
	Response *http.Response
	// Error is the error to return
	Error error
	// Handler is a function that can handle requests dynamically
	Handler func(*http.Request) (*http.Response, error)
}

// RoundTrip implements http.RoundTripper.
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	if m.Handler != nil {
		return m.Handler(req)
	}

	if m.Response != nil {
		return m.Response, nil
	}

	// Default: return 200 OK with empty body
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
	}, nil
}

// NewMockHTTPClient creates a new HTTP client with a mock transport.
func NewMockHTTPClient(roundTripper http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: roundTripper,
	}
}

// NewSuccessResponse creates a mock HTTP response with the given status code and body.
func NewSuccessResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// NewErrorResponse creates a mock HTTP error response.
func NewErrorResponse(statusCode int, errorMessage string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(errorMessage)),
		Header:     make(http.Header),
	}
}

// NewJSONResponse creates a mock HTTP response with JSON content type.
func NewJSONResponse(statusCode int, jsonBody string) *http.Response {
	resp := NewSuccessResponse(statusCode, jsonBody)
	resp.Header.Set("Content-Type", "application/json")
	return resp
}
