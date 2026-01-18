// Package server provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates improved testing practices including table-driven tests,
// concurrency testing, and integration test patterns.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerCreationAdvanced provides advanced table-driven tests for server creation.
func TestServerCreationAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setup       func(t *testing.T) *AdvancedMockServer
		validate    func(t *testing.T, server *AdvancedMockServer)
		wantErr     bool
	}{
		{
			name:        "basic_server_creation",
			description: "Create basic server with minimal config",
			setup: func(t *testing.T) *AdvancedMockServer {
				server := NewAdvancedMockServer("test-server", "http", 8080)
				return server
			},
			validate: func(t *testing.T, server *AdvancedMockServer) {
				assert.NotNil(t, server)
				assert.Equal(t, "test-server", server.GetName())
				assert.Equal(t, "http", server.GetServerType())
				assert.Equal(t, 8080, server.GetPort())
				assert.False(t, server.IsRunning())
			},
		},
		{
			name:        "server_with_error_option",
			description: "Create server configured to return errors",
			setup: func(t *testing.T) *AdvancedMockServer {
				server := NewAdvancedMockServer("error-server", "http", 8081,
					WithMockError(true, errors.New("simulated error")))
				return server
			},
			validate: func(t *testing.T, server *AdvancedMockServer) {
				assert.NotNil(t, server)
				ctx := context.Background()
				err := server.Start(ctx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "simulated error")
			},
		},
		{
			name:        "server_with_response_delay",
			description: "Create server with configured response delay",
			setup: func(t *testing.T) *AdvancedMockServer {
				server := NewAdvancedMockServer("delayed-server", "http", 8082,
					WithResponseDelay(10*time.Millisecond))
				return server
			},
			validate: func(t *testing.T, server *AdvancedMockServer) {
				assert.NotNil(t, server)
				ctx := context.Background()
				require.NoError(t, server.Start(ctx))
				defer server.Stop(ctx)

				start := time.Now()
				_, duration, err := server.HandleRequest("GET", "/api/test")
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, time.Since(start), 10*time.Millisecond)
				assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
			},
		},
		{
			name:        "server_with_load_simulation",
			description: "Create server with load simulation enabled",
			setup: func(t *testing.T) *AdvancedMockServer {
				server := NewAdvancedMockServer("load-server", "http", 8083,
					WithLoadSimulation(true))
				return server
			},
			validate: func(t *testing.T, server *AdvancedMockServer) {
				assert.NotNil(t, server)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			server := tt.setup(t)
			tt.validate(t, server)
		})
	}
}

// TestServerLifecycleAdvanced tests server lifecycle operations.
func TestServerLifecycleAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "server_startup",
			description: "Test server startup sequence",
			testFunc: func(t *testing.T) {
				server := NewAdvancedMockServer("lifecycle-server", "http", 8090)
				ctx := context.Background()

				// Verify initial state
				assert.False(t, server.IsRunning())
				assert.Equal(t, 0, server.GetCallCount())

				// Start server
				err := server.Start(ctx)
				require.NoError(t, err)

				// Verify running state
				assert.True(t, server.IsRunning())
				assert.Equal(t, 1, server.GetCallCount())

				// Cleanup
				err = server.Stop(ctx)
				require.NoError(t, err)
			},
		},
		{
			name:        "server_shutdown",
			description: "Test server shutdown sequence",
			testFunc: func(t *testing.T) {
				server := NewAdvancedMockServer("shutdown-server", "http", 8091)
				ctx := context.Background()

				// Start and verify
				err := server.Start(ctx)
				require.NoError(t, err)
				assert.True(t, server.IsRunning())

				// Add some connections
				server.AddConnection()
				server.AddConnection()
				assert.Equal(t, 2, server.GetConnectionCount())

				// Shutdown
				err = server.Stop(ctx)
				require.NoError(t, err)

				// Verify shutdown state
				assert.False(t, server.IsRunning())
				assert.Equal(t, 0, server.GetConnectionCount())
			},
		},
		{
			name:        "double_start_error",
			description: "Test error when starting already running server",
			testFunc: func(t *testing.T) {
				server := NewAdvancedMockServer("double-start-server", "http", 8092)
				ctx := context.Background()

				err := server.Start(ctx)
				require.NoError(t, err)
				defer server.Stop(ctx)

				// Try to start again
				err = server.Start(ctx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "already running")
			},
		},
		{
			name:        "stop_non_running_error",
			description: "Test error when stopping non-running server",
			testFunc: func(t *testing.T) {
				server := NewAdvancedMockServer("non-running-server", "http", 8093)
				ctx := context.Background()

				// Try to stop without starting
				err := server.Stop(ctx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not running")
			},
		},
		{
			name:        "full_lifecycle",
			description: "Test complete server lifecycle with requests",
			testFunc: func(t *testing.T) {
				server := NewAdvancedMockServer("full-lifecycle-server", "http", 8094)
				ctx := context.Background()

				// Start
				err := server.Start(ctx)
				require.NoError(t, err)

				// Handle some requests
				for i := 0; i < 5; i++ {
					statusCode, _, err := server.HandleRequest("GET", "/api/test")
					assert.NoError(t, err)
					assert.Equal(t, 200, statusCode)
				}

				// Verify request history
				history := server.GetRequestHistory()
				assert.Len(t, history, 5)

				// Stop
				err = server.Stop(ctx)
				require.NoError(t, err)
				assert.False(t, server.IsRunning())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			tt.testFunc(t)
		})
	}
}

// TestServerRequestHandlingAdvanced tests various request handling scenarios.
func TestServerRequestHandlingAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		method      string
		path        string
		serverOpts  []MockServerOption
		expectError bool
		expectedMin int // minimum expected status code
		expectedMax int // maximum expected status code
	}{
		{
			name:        "get_request",
			description: "Handle GET request successfully",
			method:      "GET",
			path:        "/api/health",
			serverOpts:  nil,
			expectError: false,
			expectedMin: 200,
			expectedMax: 299,
		},
		{
			name:        "post_request",
			description: "Handle POST request successfully",
			method:      "POST",
			path:        "/api/chat",
			serverOpts:  nil,
			expectError: false,
			expectedMin: 200,
			expectedMax: 299,
		},
		{
			name:        "error_request",
			description: "Handle request with server error",
			method:      "GET",
			path:        "/api/error",
			serverOpts:  []MockServerOption{WithMockError(true, errors.New("server error"))},
			expectError: true,
			expectedMin: 500,
			expectedMax: 599,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			server := NewAdvancedMockServer("request-test-server", "http", 9000, tt.serverOpts...)
			ctx := context.Background()

			if !tt.expectError {
				err := server.Start(ctx)
				require.NoError(t, err)
				defer server.Stop(ctx)
			}

			statusCode, duration, err := server.HandleRequest(tt.method, tt.path)

			if tt.expectError {
				assert.Error(t, err)
			}

			assert.GreaterOrEqual(t, statusCode, tt.expectedMin)
			assert.LessOrEqual(t, statusCode, tt.expectedMax)
			assert.Greater(t, duration, time.Duration(0))
		})
	}
}

// TestConcurrentServerRequests tests concurrent server request handling.
func TestConcurrentServerRequests(t *testing.T) {
	const numGoroutines = 20
	const numRequestsPerGoroutine = 5

	// Create a simple test server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errChan := make(chan error, numGoroutines*numRequestsPerGoroutine)
	successCount := make(chan int, numGoroutines*numRequestsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numRequestsPerGoroutine; j++ {
				resp, err := http.Get(server.URL)
				if err != nil {
					errChan <- err
					continue
				}
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					successCount <- 1
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)
	close(successCount)

	// Count errors and successes
	errorCount := 0
	for range errChan {
		errorCount++
	}

	total := 0
	for range successCount {
		total++
	}

	// All requests should succeed
	assert.Equal(t, 0, errorCount, "No errors expected")
	assert.Equal(t, numGoroutines*numRequestsPerGoroutine, total, "All requests should succeed")
}

// TestConcurrentServerOperations tests concurrent operations on mock server.
func TestConcurrentServerOperations(t *testing.T) {
	const numGoroutines = 10
	const numOperationsPerGoroutine = 10

	server := NewAdvancedMockServer("concurrent-server", "http", 9100)
	ctx := context.Background()

	err := server.Start(ctx)
	require.NoError(t, err)
	defer server.Stop(ctx)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				switch j % 4 {
				case 0:
					server.AddConnection()
				case 1:
					server.RemoveConnection()
				case 2:
					server.HandleRequest("GET", "/api/test")
				case 3:
					server.CheckHealth()
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify server is still healthy after concurrent operations
	health := server.CheckHealth()
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health["status"])
}

// TestServerWithContext tests server operations with context.
func TestServerWithContext(t *testing.T) {
	t.Run("server_with_timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		server := NewAdvancedMockServer("context-server", "http", 9200)

		err := server.Start(ctx)
		require.NoError(t, err)
		defer server.Stop(ctx)

		// Perform request within context
		statusCode, _, err := server.HandleRequest("GET", "/api/test")
		assert.NoError(t, err)
		assert.Equal(t, 200, statusCode)
	})

	t.Run("server_operations_with_cancelled_context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		server := NewAdvancedMockServer("cancel-server", "http", 9201)

		err := server.Start(ctx)
		require.NoError(t, err)

		// Cancel context
		cancel()

		// Operations after cancellation
		// Server should still handle gracefully
		health := server.CheckHealth()
		assert.NotNil(t, health)

		// Stop with cancelled context
		server.Stop(ctx)
	})
}

// TestServerHealthCheck tests server health check functionality.
func TestServerHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setup       func() *AdvancedMockServer
		validate    func(t *testing.T, health map[string]any)
	}{
		{
			name:        "healthy_server",
			description: "Health check on healthy server",
			setup: func() *AdvancedMockServer {
				server := NewAdvancedMockServer("healthy-server", "http", 9300)
				ctx := context.Background()
				server.Start(ctx)
				return server
			},
			validate: func(t *testing.T, health map[string]any) {
				assert.Equal(t, "healthy", health["status"])
				assert.Equal(t, "healthy-server", health["name"])
				assert.Equal(t, "http", health["type"])
				assert.Equal(t, true, health["running"])
			},
		},
		{
			name:        "server_with_connections",
			description: "Health check with active connections",
			setup: func() *AdvancedMockServer {
				server := NewAdvancedMockServer("connected-server", "http", 9301)
				ctx := context.Background()
				server.Start(ctx)
				server.AddConnection()
				server.AddConnection()
				server.AddConnection()
				return server
			},
			validate: func(t *testing.T, health map[string]any) {
				assert.Equal(t, 3, health["connections"])
			},
		},
		{
			name:        "server_with_request_history",
			description: "Health check with request history",
			setup: func() *AdvancedMockServer {
				server := NewAdvancedMockServer("history-server", "http", 9302)
				ctx := context.Background()
				server.Start(ctx)
				server.HandleRequest("GET", "/api/test1")
				server.HandleRequest("POST", "/api/test2")
				return server
			},
			validate: func(t *testing.T, health map[string]any) {
				assert.Equal(t, 2, health["request_count"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			server := tt.setup()
			defer server.Stop(context.Background())

			health := server.CheckHealth()
			AssertServerHealth(t, health, "healthy")
			tt.validate(t, health)
		})
	}
}

// TestServerHandlerRegistry tests handler registration.
func TestServerHandlerRegistry(t *testing.T) {
	server := NewAdvancedMockServer("registry-server", "http", 9400)
	ctx := context.Background()

	// Register handlers
	server.RegisterHandler("/api/v1/chat", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server.RegisterHandler("/api/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server.RegisterHandler("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Verify handler count
	assert.Equal(t, 3, server.GetHandlerCount())

	// Start and test
	err := server.Start(ctx)
	require.NoError(t, err)
	defer server.Stop(ctx)

	health := server.CheckHealth()
	assert.Equal(t, 3, health["handler_count"])
}

// TestServerLoadTest runs a load test scenario.
func TestServerLoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	server := NewAdvancedMockServer("load-test-server", "http", 9500)

	// Run load test with 100 requests and 10 concurrent workers
	RunLoadTest(t, server, 100, 10)

	// Verify all requests were recorded
	history := server.GetRequestHistory()
	assert.Len(t, history, 100)
}

// TestServerIntegrationScenarios tests integration scenarios.
func TestServerIntegrationScenarios(t *testing.T) {
	t.Run("api_endpoint_scenario", func(t *testing.T) {
		server := NewAdvancedMockServer("integration-server", "http", 9600)
		runner := NewServerScenarioRunner(server)

		endpoints := []APIEndpoint{
			{Method: "GET", Path: "/api/health", ExpectError: false},
			{Method: "POST", Path: "/api/chat", ExpectError: false},
			{Method: "GET", Path: "/api/embeddings", ExpectError: false},
		}

		err := runner.RunAPIEndpointScenario(context.Background(), endpoints)
		assert.NoError(t, err)
	})

	t.Run("connection_management_scenario", func(t *testing.T) {
		server := NewAdvancedMockServer("connection-server", "http", 9601)
		runner := NewServerScenarioRunner(server)

		err := runner.RunConnectionManagementScenario(context.Background(), 50)
		assert.NoError(t, err)
	})
}

// BenchmarkServerCreation benchmarks server creation performance.
func BenchmarkServerCreation(b *testing.B) {
	b.Run("basic_creation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewAdvancedMockServer("benchmark-server", "http", 9700+i)
		}
	})

	b.Run("creation_with_options", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewAdvancedMockServer("benchmark-server", "http", 9700+i,
				WithResponseDelay(1*time.Millisecond),
				WithLoadSimulation(true))
		}
	})
}

// BenchmarkServerLifecycle benchmarks server lifecycle operations.
func BenchmarkServerLifecycle(b *testing.B) {
	server := NewAdvancedMockServer("benchmark-lifecycle-server", "http", 9800)
	helper := NewBenchmarkHelper(server, 10)

	duration, err := helper.BenchmarkServerLifecycle(b.N)
	require.NoError(b, err)

	b.ReportMetric(float64(b.N)/duration.Seconds(), "lifecycles/sec")
}

// BenchmarkServerRequests benchmarks server request handling performance.
func BenchmarkServerRequests(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkMockServerRequests benchmarks mock server request handling.
func BenchmarkMockServerRequests(b *testing.B) {
	server := NewAdvancedMockServer("benchmark-mock-server", "http", 9900)
	ctx := context.Background()
	server.Start(ctx)
	defer server.Stop(ctx)

	helper := NewBenchmarkHelper(server, 10)

	duration, err := helper.BenchmarkRequestHandling(b.N)
	require.NoError(b, err)

	b.ReportMetric(float64(b.N)/duration.Seconds(), "requests/sec")
}

// TestDefaultRESTConfig tests the DefaultRESTConfig function.
func TestDefaultRESTConfig(t *testing.T) {
	config := DefaultRESTConfig()

	// Verify base config values
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 8080, config.Port)
	assert.Equal(t, 30*time.Second, config.ReadTimeout)
	assert.Equal(t, 30*time.Second, config.WriteTimeout)
	assert.Equal(t, 120*time.Second, config.IdleTimeout)
	assert.Equal(t, 1<<20, config.MaxHeaderBytes) // 1MB
	assert.True(t, config.EnableCORS)
	assert.Contains(t, config.CORSOrigins, "*")
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.EnableTracing)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, 30*time.Second, config.ShutdownTimeout)

	// Verify REST-specific config values
	assert.Equal(t, "/api/v1", config.APIBasePath)
	assert.True(t, config.EnableStreaming)
	assert.Equal(t, int64(10<<20), config.MaxRequestSize) // 10MB
	assert.Equal(t, 1000, config.RateLimitRequests)
	assert.True(t, config.EnableRateLimit)
}

// TestDefaultMCPConfig tests the DefaultMCPConfig function.
func TestDefaultMCPConfig(t *testing.T) {
	config := DefaultMCPConfig()

	// Verify base config values
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 8081, config.Port)
	assert.Equal(t, 30*time.Second, config.ReadTimeout)
	assert.Equal(t, 30*time.Second, config.WriteTimeout)
	assert.Equal(t, 120*time.Second, config.IdleTimeout)
	assert.Equal(t, 1<<20, config.MaxHeaderBytes) // 1MB
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.EnableTracing)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, 30*time.Second, config.ShutdownTimeout)

	// Verify MCP-specific config values
	assert.Equal(t, "beluga-mcp-server", config.ServerName)
	assert.Equal(t, "1.0.0", config.ServerVersion)
	assert.Equal(t, "2024-11-05", config.ProtocolVersion)
	assert.Equal(t, 10, config.MaxConcurrentRequests)
	assert.Equal(t, 60*time.Second, config.RequestTimeout)
}

// TestCORSMiddleware tests the CORSMiddleware function.
func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		requestOrigin  string
		method         string
		expectCORS     bool
		expectHeaders  map[string]string
	}{
		{
			name:           "wildcard_origin_allowed",
			allowedOrigins: []string{"*"},
			requestOrigin:  "https://example.com",
			method:         "GET",
			expectCORS:     true,
			expectHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://example.com",
			},
		},
		{
			name:           "specific_origin_allowed",
			allowedOrigins: []string{"https://example.com", "https://test.com"},
			requestOrigin:  "https://example.com",
			method:         "POST",
			expectCORS:     true,
			expectHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://example.com",
			},
		},
		{
			name:           "origin_not_allowed",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "https://unauthorized.com",
			method:         "GET",
			expectCORS:     false,
		},
		{
			name:           "no_origin_header",
			allowedOrigins: []string{"*"},
			requestOrigin:  "",
			method:         "GET",
			expectCORS:     false,
		},
		{
			name:           "options_preflight",
			allowedOrigins: []string{"*"},
			requestOrigin:  "https://example.com",
			method:         "OPTIONS",
			expectCORS:     true,
			expectHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := CORSMiddleware(tt.allowedOrigins)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if tt.expectCORS {
				for header, expectedValue := range tt.expectHeaders {
					assert.Equal(t, expectedValue, resp.Header.Get(header), "CORS header %s should match", header)
				}
				assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "GET")
				assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Content-Type")
			} else {
				assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"), "CORS headers should not be set")
			}

			if tt.method == "OPTIONS" {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
}

// TestLoggingMiddleware tests the LoggingMiddleware function.
func TestLoggingMiddleware(t *testing.T) {
	logger := newMockLogger()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := LoggingMiddleware(logger)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Verify logging occurred
	logs := logger.getLogs("INFO")
	assert.Greater(t, len(logs), 0, "Should log request")

	// Check that log contains expected fields
	found := false
	for _, log := range logs {
		if strings.Contains(log.message, "HTTP Request") {
			found = true
			// Verify log args contain expected fields
			argsStr := fmt.Sprintf("%v", log.args)
			assert.Contains(t, argsStr, "GET", "Should log method")
			assert.Contains(t, argsStr, "/test", "Should log path")
			break
		}
	}
	assert.True(t, found, "Should find HTTP Request log entry")
}

// TestRecoveryMiddleware tests the RecoveryMiddleware function.
func TestRecoveryMiddleware(t *testing.T) {
	logger := newMockLogger()

	tests := []struct {
		name        string
		handler     http.HandlerFunc
		expectPanic bool
		expectError bool
	}{
		{
			name: "normal_handler",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectPanic: false,
			expectError: false,
		},
		{
			name: "handler_with_panic",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("test panic")
			},
			expectPanic: true,
			expectError: true,
		},
		{
			name: "handler_with_string_panic",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("string panic")
			},
			expectPanic: true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := RecoveryMiddleware(logger)
			wrappedHandler := middleware(tt.handler)

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			// Should not panic even if handler panics
			assert.NotPanics(t, func() {
				wrappedHandler.ServeHTTP(w, req)
			})

			resp := w.Result()
			defer resp.Body.Close()

			if tt.expectPanic {
				assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
				// Verify error was logged
				errorLogs := logger.getLogs("ERROR")
				assert.Greater(t, len(errorLogs), 0, "Should log panic recovery")
			} else {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
}

// TestIsServerError tests the IsServerError function.
func TestIsServerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "server_error",
			err:      NewInvalidRequestError("test_op", "test message", nil),
			expected: true,
		},
		{
			name:     "not_server_error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil_error",
			err:      nil,
			expected: false,
		},
		{
			name:     "wrapped_server_error",
			err:      fmt.Errorf("wrapped: %w", NewInternalError("test_op", errors.New("underlying"))),
			expected: true, // errors.As works with wrapped errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsServerError(tt.err)
			assert.Equal(t, tt.expected, result, "IsServerError should return %v for %v", tt.expected, tt.err)
		})
	}
}

// TestAsServerError tests the AsServerError function.
func TestAsServerError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedFound bool
		expectedCode  string
	}{
		{
			name:          "server_error",
			err:           NewInvalidRequestError("test_op", "test message", nil),
			expectedFound: true,
			expectedCode:  string(ErrCodeInvalidRequest),
		},
		{
			name:          "not_server_error",
			err:           errors.New("regular error"),
			expectedFound: false,
		},
		{
			name:          "nil_error",
			err:           nil,
			expectedFound: false,
		},
		{
			name:          "internal_error",
			err:           NewInternalError("test_op", errors.New("underlying")),
			expectedFound: true,
			expectedCode:  string(ErrCodeInternalError),
		},
		{
			name:          "not_found_error",
			err:           NewNotFoundError("test_op", "resource"),
			expectedFound: true,
			expectedCode:  string(ErrCodeNotFound),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverErr, found := AsServerError(tt.err)
			assert.Equal(t, tt.expectedFound, found, "AsServerError should find error: %v", tt.expectedFound)

			if tt.expectedFound {
				assert.NotNil(t, serverErr, "ServerError should not be nil")
				assert.Equal(t, tt.expectedCode, serverErr.Code, "Error code should match")
			} else {
				assert.Nil(t, serverErr, "ServerError should be nil when not found")
			}
		})
	}
}
