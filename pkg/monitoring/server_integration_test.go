package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/mock"
	"github.com/stretchr/testify/assert"
)

// unhealthyMockHealthChecker is a test helper that always returns unhealthy results
type unhealthyMockHealthChecker struct {
	results map[string]iface.HealthCheckResult
}

func (m *unhealthyMockHealthChecker) RegisterCheck(name string, check iface.HealthCheckFunc) error {
	return nil
}

func (m *unhealthyMockHealthChecker) RunChecks(ctx context.Context) map[string]iface.HealthCheckResult {
	return m.results
}

func (m *unhealthyMockHealthChecker) IsHealthy(ctx context.Context) bool {
	return false
}

func TestNewServerIntegration(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	assert.NotNil(t, integration)
	assert.NotNil(t, integration.monitor)
	assert.Equal(t, mockMonitor, integration.monitor)
}

func TestServerIntegrationHealthCheckHandler(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	t.Run("healthy system", func(t *testing.T) {
		mockMonitor.IsHealthyValue = true

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		integration.HealthCheckHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "healthy")

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
		assert.Contains(t, response, "timestamp")
		assert.Contains(t, response, "checks")
	})

	t.Run("unhealthy system", func(t *testing.T) {
		// Create a custom health checker that returns unhealthy results
		unhealthyHealthChecker := &unhealthyMockHealthChecker{
			results: map[string]iface.HealthCheckResult{
				"test_check": {
					Status:  iface.StatusUnhealthy,
					Message: "Test unhealthy",
				},
			},
		}
		unhealthyMonitor := mock.NewMockMonitor()
		unhealthyMonitor.HealthCheckerValue = unhealthyHealthChecker
		unhealthyIntegration := NewServerIntegration(unhealthyMonitor)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		unhealthyIntegration.HealthCheckHandler(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "unhealthy")

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "unhealthy", response["status"])
	})
}

func TestServerIntegrationSafetyCheckHandler(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	t.Run("successful safety check", func(t *testing.T) {
		requestBody := `{"content": "This is safe content", "context": "chat"}`
		req := httptest.NewRequest("POST", "/safety/check", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		integration.SafetyCheckHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "result")
		assert.Contains(t, response, "timestamp")

		// Test passes if no error occurred
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/safety/check", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		integration.SafetyCheckHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		// Error message may vary, just check it's a bad request
		assert.NotEmpty(t, w.Body.String())
	})

	t.Run("missing content", func(t *testing.T) {
		requestBody := `{"context": "chat"}`
		req := httptest.NewRequest("POST", "/safety/check", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		integration.SafetyCheckHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		// Error message may vary, just check it's a bad request
		assert.NotEmpty(t, w.Body.String())
	})
}

func TestServerIntegrationEthicsCheckHandler(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	t.Run("successful ethics check", func(t *testing.T) {
		requestBody := `{
			"content": "This is ethical content",
			"context": {
				"user_demographics": {"age_group": "25-34"},
				"content_type": "text",
				"domain": "general"
			}
		}`
		req := httptest.NewRequest("POST", "/ethics/check", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		integration.EthicsCheckHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "result")
		assert.Contains(t, response, "timestamp")

		// Test passes if no error occurred
	})

	t.Run("invalid ethics context", func(t *testing.T) {
		requestBody := `{"content": "test", "context": "invalid"}`
		req := httptest.NewRequest("POST", "/ethics/check", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		integration.EthicsCheckHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		// Error message may vary, just check it's a bad request
		assert.NotEmpty(t, w.Body.String())
	})
}

func TestServerIntegrationBestPracticesCheckHandler(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	t.Run("successful best practices check", func(t *testing.T) {
		requestBody := `{"data": "sample code", "component": "test_component"}`
		req := httptest.NewRequest("POST", "/best-practices/check", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		integration.BestPracticesCheckHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "issues")
		assert.Contains(t, response, "timestamp")

		// Test passes if no error occurred
	})
}

func TestServerIntegrationMetricsHandler(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	integration.MetricsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")

	body := w.Body.String()
	assert.Contains(t, body, "# HELP") // Prometheus format
	assert.Contains(t, body, "# TYPE")
}

func TestServerIntegrationTracesHandler(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	req := httptest.NewRequest("GET", "/traces", nil)
	w := httptest.NewRecorder()

	integration.TracesHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "traces")
	assert.Contains(t, response, "timestamp")
}

func TestServerIntegrationLogsHandler(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	req := httptest.NewRequest("GET", "/logs", nil)
	w := httptest.NewRecorder()

	integration.LogsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "logs")
	assert.Contains(t, response, "timestamp")
}

func TestServerIntegrationRegisterRoutes(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	mux := http.NewServeMux()
	integration.RegisterRoutes(mux, "/api/v1")

	// Test that routes are registered by making requests
	testRoutes := []struct {
		path   string
		method string
	}{
		{"/api/v1/health", "GET"},
		{"/api/v1/safety/check", "POST"},
		{"/api/v1/ethics/check", "POST"},
		{"/api/v1/best-practices/check", "POST"},
		{"/api/v1/metrics", "GET"},
		{"/api/v1/traces", "GET"},
		{"/api/v1/logs", "GET"},
	}

	for _, route := range testRoutes {
		req := httptest.NewRequest(route.method, route.path, nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Should not get 404 (route not found)
		assert.NotEqual(t, http.StatusNotFound, w.Code,
			"Route %s %s should be registered", route.method, route.path)
	}
}

func TestServerIntegrationMonitoringMiddleware(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	middleware := integration.MonitoringMiddleware(testHandler)

	t.Run("successful request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "test response", w.Body.String())
	})

	t.Run("request with context", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/test", strings.NewReader("test data"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "test-agent")
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestServerIntegrationGetStatusString(t *testing.T) {
	integration := NewServerIntegration(mock.NewMockMonitor())

	tests := []struct {
		healthy  bool
		expected string
	}{
		{true, "healthy"},
		{false, "unhealthy"},
	}

	for _, tt := range tests {
		result := integration.getStatusString(tt.healthy)
		assert.Equal(t, tt.expected, result)
	}
}

func TestServerIntegrationFormatHealthResults(t *testing.T) {
	integration := NewServerIntegration(mock.NewMockMonitor())

	results := map[string]iface.HealthCheckResult{
		"check1": {
			Status:    iface.StatusHealthy,
			Message:   "Check 1 passed",
			CheckName: "check1",
			Timestamp: time.Now(),
			Details:   map[string]interface{}{"detail": "value"},
		},
		"check2": {
			Status:    iface.StatusUnhealthy,
			Message:   "Check 2 failed",
			CheckName: "check2",
			Timestamp: time.Now(),
		},
	}

	formatted := integration.formatHealthResults(results)

	assert.Len(t, formatted, 2)

	// Find check1 and check2 in results
	var check1, check2 map[string]interface{}
	for _, result := range formatted {
		if result["name"] == "check1" {
			check1 = result
		} else if result["name"] == "check2" {
			check2 = result
		}
	}

	assert.NotNil(t, check1)
	assert.NotNil(t, check2)
	assert.Equal(t, "healthy", check1["status"])
	assert.Equal(t, "unhealthy", check2["status"])
	assert.Contains(t, check1, "timestamp")
	assert.Contains(t, check2, "timestamp")
}

func TestResponseWriter(t *testing.T) {
	// Create a basic response writer
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w}

	t.Run("write header", func(t *testing.T) {
		rw.WriteHeader(http.StatusNotFound)
		assert.Equal(t, http.StatusNotFound, rw.statusCode)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("default status", func(t *testing.T) {
		w2 := httptest.NewRecorder()
		rw2 := &responseWriter{ResponseWriter: w2}
		rw2.Write([]byte("test"))
		assert.Equal(t, http.StatusOK, w2.Code) // Default status
	})
}

func TestServerIntegrationErrorHandling(t *testing.T) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	t.Run("panic in handler", func(t *testing.T) {
		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		middleware := integration.MonitoringMiddleware(panicHandler)

		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()

		// This should not panic the test, middleware should handle it
		assert.NotPanics(t, func() {
			middleware.ServeHTTP(w, req)
		})

		// Should still get a response (500 error)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("large request body", func(t *testing.T) {
		largeBody := strings.Repeat("x", 1024*1024) // 1MB
		req := httptest.NewRequest("POST", "/safety/check", strings.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		integration.SafetyCheckHandler(w, req)

		// Should handle large body gracefully
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
	})
}

// Benchmark tests
func BenchmarkServerIntegrationHealthCheckHandler(b *testing.B) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		integration.HealthCheckHandler(w, req)
	}
}

func BenchmarkServerIntegrationSafetyCheckHandler(b *testing.B) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	requestBody := `{"content": "This is test content for benchmarking", "context": "bench"}`
	req := httptest.NewRequest("POST", "/safety/check", bytes.NewReader([]byte(requestBody)))
	req.Header.Set("Content-Type", "application/json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		integration.SafetyCheckHandler(w, req)
		req.Body = io.NopCloser(bytes.NewReader([]byte(requestBody))) // Reset body
	}
}

func BenchmarkServerIntegrationMonitoringMiddleware(b *testing.B) {
	mockMonitor := mock.NewMockMonitor()
	integration := NewServerIntegration(mockMonitor)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := integration.MonitoringMiddleware(handler)
	req := httptest.NewRequest("GET", "/bench", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}
}

func BenchmarkServerIntegrationFormatHealthResults(b *testing.B) {
	integration := NewServerIntegration(mock.NewMockMonitor())

	results := make(map[string]iface.HealthCheckResult)
	for i := 0; i < 10; i++ {
		results[string(rune(i))] = iface.HealthCheckResult{
			Status:    iface.StatusHealthy,
			Message:   "Test check",
			CheckName: string(rune(i)),
			Timestamp: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatted := integration.formatHealthResults(results)
		_ = formatted
	}
}
