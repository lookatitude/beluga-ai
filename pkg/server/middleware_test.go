// Package server provides comprehensive tests for middleware functionality.
// These tests cover CORS, logging, recovery, and custom middleware chains.
package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Middleware Chain Tests

func TestMiddlewareChain(t *testing.T) {
	logger := newMockLogger()

	// Create a chain of middlewares
	middlewares := []Middleware{
		CORSMiddleware([]string{"http://example.com", "*"}),
		LoggingMiddleware(logger),
		RecoveryMiddleware(logger),
	}

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status": "success"}`)
	})

	// Apply middlewares in reverse order (last middleware wraps first)
	var wrappedHandler http.Handler = testHandler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrappedHandler = middlewares[i](wrappedHandler)
	}

	// Test the middleware chain
	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify CORS headers
	corsHeader := w.Header().Get("Access-Control-Allow-Origin")
	if corsHeader != "http://example.com" {
		t.Errorf("Expected CORS origin 'http://example.com', got '%s'", corsHeader)
	}

	// Verify logging occurred
	if !logger.hasLog("INFO", "HTTP Request") {
		t.Error("Expected HTTP request to be logged")
	}
}

func TestCORSMiddlewareWithMultipleOrigins(t *testing.T) {
	tests := []struct {
		name           string
		requestOrigin  string
		expectedOrigin string
		allowedOrigins []string
		expectAllow    bool
	}{
		{
			name:           "exact_match",
			allowedOrigins: []string{"http://example.com", "http://test.com"},
			requestOrigin:  "http://example.com",
			expectAllow:    true,
			expectedOrigin: "http://example.com",
		},
		{
			name:           "wildcard_match",
			allowedOrigins: []string{"*", "http://test.com"},
			requestOrigin:  "http://example.com",
			expectAllow:    true,
			expectedOrigin: "http://example.com",
		},
		{
			name:           "no_match",
			allowedOrigins: []string{"http://test.com", "http://other.com"},
			requestOrigin:  "http://example.com",
			expectAllow:    false,
			expectedOrigin: "",
		},
		{
			name:           "only_wildcard",
			allowedOrigins: []string{"*"},
			requestOrigin:  "http://any-domain.com",
			expectAllow:    true,
			expectedOrigin: "http://any-domain.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := CORSMiddleware(tt.allowedOrigins)
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			actualOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectAllow && actualOrigin == "" {
				t.Error("Expected CORS to be allowed")
			}
			if !tt.expectAllow && actualOrigin != "" {
				t.Errorf("Expected CORS to be denied, but got origin: %s", actualOrigin)
			}
			if tt.expectAllow && tt.expectedOrigin != "" && actualOrigin != tt.expectedOrigin && !contains(tt.allowedOrigins, "*") {
				t.Errorf("Expected origin %s, got %s", tt.expectedOrigin, actualOrigin)
			}
		})
	}
}

func TestLoggingMiddlewareWithErrors(t *testing.T) {
	logger := newMockLogger()
	middleware := LoggingMiddleware(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/error", http.NoBody)
	req.Header.Set("User-Agent", "error-client")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	if !logger.hasLog("INFO", "HTTP Request") {
		t.Error("Expected HTTP request to be logged even with error")
	}
}

func TestRecoveryMiddlewarePanic(t *testing.T) {
	logger := newMockLogger()
	middleware := RecoveryMiddleware(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic message")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/panic", http.NoBody)
	w := httptest.NewRecorder()

	// This should not panic
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for panic, got %d", w.Code)
	}

	if !logger.hasLog("ERROR", "Panic recovered") {
		t.Error("Expected panic to be logged")
	}
}

func TestRecoveryMiddlewareNoPanic(t *testing.T) {
	logger := newMockLogger()
	middleware := RecoveryMiddleware(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "success")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/normal", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should not log any panic messages
	if logger.hasLog("ERROR", "Panic recovered") {
		t.Error("Unexpected panic log for normal operation")
	}
}

func TestCustomMiddleware(t *testing.T) {
	// Test custom middleware that adds custom headers
	customMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "custom-value")
			w.Header().Set("X-Request-Path", r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}

	logger := newMockLogger()
	middlewareChain := []Middleware{
		CORSMiddleware([]string{"*"}),
		LoggingMiddleware(logger),
		customMiddleware,
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "response body")
	})

	// Apply middleware chain
	var wrappedHandler http.Handler = baseHandler
	for i := len(middlewareChain) - 1; i >= 0; i-- {
		wrappedHandler = middlewareChain[i](wrappedHandler)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/custom", http.NoBody)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Verify custom headers
	customHeader := w.Header().Get("X-Custom-Header")
	if customHeader != "custom-value" {
		t.Errorf("Expected custom header 'custom-value', got '%s'", customHeader)
	}

	pathHeader := w.Header().Get("X-Request-Path")
	if pathHeader != "/api/custom" {
		t.Errorf("Expected path header '/api/custom', got '%s'", pathHeader)
	}

	// Verify CORS still works
	corsHeader := w.Header().Get("Access-Control-Allow-Origin")
	if corsHeader != "http://example.com" {
		t.Errorf("Expected CORS origin 'http://example.com', got '%s'", corsHeader)
	}

	// Verify logging still works
	if !logger.hasLog("INFO", "HTTP Request") {
		t.Error("Expected HTTP request to be logged")
	}
}

func TestMiddlewareOrder(t *testing.T) {
	var executionOrder []string

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware1-start")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware1-end")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware2-start")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware2-end")
		})
	}

	middleware3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware3-start")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware3-end")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	// Apply middlewares (order matters!)
	wrappedHandler := middleware1(middleware2(middleware3(handler)))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Verify execution order
	expectedOrder := []string{
		"middleware1-start",
		"middleware2-start",
		"middleware3-start",
		"handler",
		"middleware3-end",
		"middleware2-end",
		"middleware1-end",
	}

	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("Expected %d execution steps, got %d", len(expectedOrder), len(executionOrder))
		return
	}

	for i, expected := range expectedOrder {
		var actual string
		if i < len(executionOrder) {
			actual = executionOrder[i]
		} else {
			actual = "out of bounds"
		}
		if executionOrder[i] != expected {
			t.Errorf("Expected execution order[%d] = %s, got %s", i, expected, actual)
		}
	}
}

// Benchmark tests for middleware

func BenchmarkCORSMiddleware(b *testing.B) {
	middleware := CORSMiddleware([]string{"http://example.com", "*"})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkLoggingMiddleware(b *testing.B) {
	logger := newMockLogger()
	middleware := LoggingMiddleware(logger)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.Header.Set("User-Agent", "bench-agent")
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkRecoveryMiddleware(b *testing.B) {
	logger := newMockLogger()
	middleware := RecoveryMiddleware(logger)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkMiddlewareChain3(b *testing.B) {
	logger := newMockLogger()
	middlewares := []Middleware{
		CORSMiddleware([]string{"*"}),
		LoggingMiddleware(logger),
		RecoveryMiddleware(logger),
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply middlewares
	var handler http.Handler = baseHandler
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("User-Agent", "bench-agent")
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkMiddlewareChain5(b *testing.B) {
	logger := newMockLogger()
	middlewares := []Middleware{
		CORSMiddleware([]string{"*"}),
		LoggingMiddleware(logger),
		RecoveryMiddleware(logger),
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Custom", "value")
				next.ServeHTTP(w, r)
			})
		},
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Another", "value2")
				next.ServeHTTP(w, r)
			})
		},
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply middlewares
	var handler http.Handler = baseHandler
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("User-Agent", "bench-agent")
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
