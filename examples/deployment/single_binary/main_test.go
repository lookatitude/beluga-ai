// Package main provides comprehensive tests for the single binary deployment example.
// These tests verify health checks, graceful shutdown, configuration, and HTTP handlers.
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Configuration Tests
// =============================================================================

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected func(*Config)
	}{
		{
			name:    "default values",
			envVars: map[string]string{},
			expected: func(c *Config) {
				assert.Equal(t, "8080", c.Port)
				assert.Equal(t, "9090", c.MetricsPort)
				assert.Equal(t, 30*time.Second, c.ShutdownTimeout)
				assert.Equal(t, "openai", c.LLMProvider)
				assert.Equal(t, "gpt-4", c.LLMModel)
			},
		},
		{
			name: "custom port",
			envVars: map[string]string{
				"PORT":         "9000",
				"METRICS_PORT": "9001",
			},
			expected: func(c *Config) {
				assert.Equal(t, "9000", c.Port)
				assert.Equal(t, "9001", c.MetricsPort)
			},
		},
		{
			name: "custom LLM settings",
			envVars: map[string]string{
				"LLM_PROVIDER": "anthropic",
				"LLM_MODEL":    "claude-3",
				"LLM_TIMEOUT":  "60s",
			},
			expected: func(c *Config) {
				assert.Equal(t, "anthropic", c.LLMProvider)
				assert.Equal(t, "claude-3", c.LLMModel)
				assert.Equal(t, 60*time.Second, c.LLMTimeout)
			},
		},
		{
			name: "observability settings",
			envVars: map[string]string{
				"OTEL_ENDPOINT":     "localhost:4317",
				"OTEL_SERVICE_NAME": "test-service",
				"LOG_LEVEL":         "debug",
				"LOG_FORMAT":        "text",
			},
			expected: func(c *Config) {
				assert.Equal(t, "localhost:4317", c.OTELEndpoint)
				assert.Equal(t, "test-service", c.ServiceName)
				assert.Equal(t, "debug", c.LogLevel)
				assert.Equal(t, "text", c.LogFormat)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars
			envKeys := []string{
				"PORT", "METRICS_PORT", "SHUTDOWN_TIMEOUT",
				"OTEL_ENDPOINT", "OTEL_SERVICE_NAME", "LOG_LEVEL", "LOG_FORMAT",
				"LLM_PROVIDER", "LLM_MODEL", "LLM_TIMEOUT",
			}
			for _, key := range envKeys {
				os.Unsetenv(key)
			}

			// Set test env vars
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			config := LoadConfig()
			tt.expected(config)
		})
	}
}

func TestGetEnv(t *testing.T) {
	// Test default value
	result := getEnv("NONEXISTENT_VAR", "default")
	assert.Equal(t, "default", result)

	// Test existing value
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result = getEnv("TEST_VAR", "default")
	assert.Equal(t, "test_value", result)
}

func TestGetDurationEnv(t *testing.T) {
	// Test default value
	result := getDurationEnv("NONEXISTENT_DURATION", 10*time.Second)
	assert.Equal(t, 10*time.Second, result)

	// Test valid duration
	os.Setenv("TEST_DURATION", "5m")
	defer os.Unsetenv("TEST_DURATION")

	result = getDurationEnv("TEST_DURATION", 10*time.Second)
	assert.Equal(t, 5*time.Minute, result)

	// Test invalid duration (should return default)
	os.Setenv("TEST_DURATION", "invalid")
	result = getDurationEnv("TEST_DURATION", 10*time.Second)
	assert.Equal(t, 10*time.Second, result)
}

// =============================================================================
// Health Check Tests
// =============================================================================

func TestHealthEndpoints(t *testing.T) {
	app := createTestApp(t)

	tests := []struct {
		name           string
		endpoint       string
		healthy        bool
		ready          bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "liveness healthy",
			endpoint:       "/health/live",
			healthy:        true,
			ready:          true,
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "liveness unhealthy",
			endpoint:       "/health/live",
			healthy:        false,
			ready:          true,
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "NOT OK",
		},
		{
			name:           "readiness ready",
			endpoint:       "/health/ready",
			healthy:        true,
			ready:          true,
			expectedStatus: http.StatusOK,
			expectedBody:   "READY",
		},
		{
			name:           "readiness not ready",
			endpoint:       "/health/ready",
			healthy:        true,
			ready:          false,
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "NOT READY",
		},
		{
			name:           "startup ready",
			endpoint:       "/health/startup",
			healthy:        true,
			ready:          true,
			expectedStatus: http.StatusOK,
			expectedBody:   "READY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.healthy.Store(tt.healthy)
			app.ready.Store(tt.ready)

			req := httptest.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()

			switch tt.endpoint {
			case "/health/live":
				app.handleLiveness(w, req)
			case "/health/ready":
				app.handleReadiness(w, req)
			case "/health/startup":
				app.handleStartup(w, req)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}

// =============================================================================
// HTTP Handler Tests
// =============================================================================

func TestHandleRoot(t *testing.T) {
	app := createTestApp(t)

	t.Run("root path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		app.handleRoot(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, app.config.ServiceName, response["service"])
		assert.Equal(t, "running", response["status"])
	})

	t.Run("not found path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()

		app.handleRoot(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleChat(t *testing.T) {
	app := createTestApp(t)
	app.ready.Store(true)

	t.Run("missing query parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/chat", nil)
		w := httptest.NewRecorder()

		app.handleChat(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not ready", func(t *testing.T) {
		app.ready.Store(false)
		defer app.ready.Store(true)

		req := httptest.NewRequest("GET", "/chat?q=hello", nil)
		w := httptest.NewRecorder()

		app.handleChat(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("mock response when no LLM", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/chat?q=hello", nil)
		w := httptest.NewRecorder()

		app.handleChat(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response["response"], "hello")
		assert.Contains(t, response["response"], "no LLM configured")
	})
}

// =============================================================================
// Active Request Tracking Tests
// =============================================================================

func TestActiveRequestTracking(t *testing.T) {
	app := createTestApp(t)
	app.ready.Store(true)

	// Verify initial count
	assert.Equal(t, int64(0), app.activeRequests.Load())

	// Simulate concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/chat?q=test", nil)
			w := httptest.NewRecorder()

			// Check active count increased during request
			go func() {
				time.Sleep(5 * time.Millisecond)
				assert.GreaterOrEqual(t, app.activeRequests.Load(), int64(1))
			}()

			app.handleChat(w, req)
		}()
	}

	wg.Wait()

	// Verify count returns to zero
	assert.Equal(t, int64(0), app.activeRequests.Load())
}

// =============================================================================
// Lifecycle Tests
// =============================================================================

func TestAppStartup(t *testing.T) {
	app := createTestApp(t)

	// Verify not ready initially
	assert.False(t, app.ready.Load())

	ctx := context.Background()
	err := app.Start(ctx)
	require.NoError(t, err)

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	// Verify ready after start
	assert.True(t, app.ready.Load())
	assert.True(t, app.healthy.Load())

	// Shutdown
	err = app.Shutdown(ctx)
	require.NoError(t, err)
}

func TestGracefulShutdown(t *testing.T) {
	app := createTestApp(t)

	ctx := context.Background()
	err := app.Start(ctx)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.True(t, app.ready.Load())

	// Start shutdown
	err = app.Shutdown(ctx)
	require.NoError(t, err)

	// Verify not ready after shutdown
	assert.False(t, app.ready.Load())
}

func TestShutdownWithActiveRequests(t *testing.T) {
	app := createTestApp(t)
	app.config.ShutdownTimeout = 5 * time.Second

	ctx := context.Background()
	err := app.Start(ctx)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Simulate active request
	app.activeRequests.Add(1)
	defer app.activeRequests.Add(-1)

	// Shutdown should still complete within timeout
	done := make(chan error, 1)
	go func() {
		done <- app.Shutdown(ctx)
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(app.config.ShutdownTimeout + time.Second):
		t.Fatal("Shutdown timed out")
	}
}

// =============================================================================
// Logging Tests
// =============================================================================

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected int // slog.Level value
	}{
		{"debug", -4},  // slog.LevelDebug
		{"info", 0},    // slog.LevelInfo
		{"warn", 4},    // slog.LevelWarn
		{"warning", 4}, // slog.LevelWarn
		{"error", 8},   // slog.LevelError
		{"unknown", 0}, // default to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := parseLogLevel(tt.input)
			assert.Equal(t, tt.expected, int(level))
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkHandleChat(b *testing.B) {
	app := createBenchApp(b)
	app.ready.Store(true)

	req := httptest.NewRequest("GET", "/chat?q=benchmark", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.handleChat(w, req)
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	app := createBenchApp(b)
	app.healthy.Store(true)
	app.ready.Store(true)

	req := httptest.NewRequest("GET", "/health/ready", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.handleReadiness(w, req)
	}
}

func BenchmarkConcurrentRequests(b *testing.B) {
	app := createBenchApp(b)
	app.ready.Store(true)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/chat?q=concurrent", nil)
			w := httptest.NewRecorder()
			app.handleChat(w, req)
		}
	})
}

// =============================================================================
// Test Helpers
// =============================================================================

func createTestApp(t *testing.T) *App {
	t.Helper()

	config := &Config{
		Port:            "0", // Use random port
		MetricsPort:     "0",
		ShutdownTimeout: 5 * time.Second,
		ServiceName:     "test-service",
		LogLevel:        "error", // Quiet during tests
		LogFormat:       "text",
		LLMProvider:     "mock",
		LLMModel:        "mock-model",
	}

	app := &App{
		config: config,
	}
	app.initLogger()

	// Initialize minimal observability
	app.tracer = nil // Skip tracing in tests
	app.initHTTPServers()

	app.healthy.Store(true)

	return app
}

func createBenchApp(b *testing.B) *App {
	b.Helper()

	config := &Config{
		Port:            "0",
		MetricsPort:     "0",
		ShutdownTimeout: 5 * time.Second,
		ServiceName:     "bench-service",
		LogLevel:        "error",
		LogFormat:       "text",
		LLMProvider:     "mock",
		LLMModel:        "mock-model",
	}

	app := &App{
		config: config,
	}
	app.initLogger()
	app.initHTTPServers()
	app.healthy.Store(true)

	return app
}
