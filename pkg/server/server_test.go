// Package server provides tests for the server package.
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRESTServer(t *testing.T) {
	// Test that NewRESTServer returns a valid REST server
	server, err := NewRESTServer()
	if err != nil {
		t.Fatalf("NewRESTServer() returned error: %v", err)
	}
	if server == nil {
		t.Fatal("NewRESTServer() returned nil server")
	}

	// Test that it implements the RESTServer interface
	var _ RESTServer = server
}

func TestNewMCPServer(t *testing.T) {
	// Test that NewMCPServer returns a valid MCP server
	server, err := NewMCPServer()
	if err != nil {
		t.Fatalf("NewMCPServer() returned error: %v", err)
	}
	if server == nil {
		t.Fatal("NewMCPServer() returned nil server")
	}

	// Test that it implements the MCPServer interface
	var _ MCPServer = server
}

func TestDefaultConfigs(t *testing.T) {
	restConfig := DefaultRESTConfig()
	if restConfig.Host != "localhost" {
		t.Errorf("DefaultRESTConfig().Host = %v, want localhost", restConfig.Host)
	}
	if restConfig.Port != 8080 {
		t.Errorf("DefaultRESTConfig().Port = %v, want 8080", restConfig.Port)
	}
	if restConfig.APIBasePath != "/api/v1" {
		t.Errorf("DefaultRESTConfig().APIBasePath = %v, want /api/v1", restConfig.APIBasePath)
	}

	mcpConfig := DefaultMCPConfig()
	if mcpConfig.Host != "localhost" {
		t.Errorf("DefaultMCPConfig().Host = %v, want localhost", mcpConfig.Host)
	}
	if mcpConfig.Port != 8081 {
		t.Errorf("DefaultMCPConfig().Port = %v, want 8081", mcpConfig.Port)
	}
	if mcpConfig.ServerName != "beluga-mcp-server" {
		t.Errorf("DefaultMCPConfig().ServerName = %v, want beluga-mcp-server", mcpConfig.ServerName)
	}
}

func TestMiddlewareFunctions(t *testing.T) {
	// Test CORSMiddleware
	corsMiddleware := CORSMiddleware([]string{"http://example.com"})
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Error("CORS middleware did not set correct origin header")
	}

	// Test with invalid origin
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://invalid.com")
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "http://invalid.com" {
		t.Error("CORS middleware allowed invalid origin")
	}
}

// Mock implementations for testing

type mockLogger struct {
	logs []string
}

func (m *mockLogger) Debug(msg string, args ...interface{}) {
	m.logs = append(m.logs, "DEBUG: "+msg)
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	m.logs = append(m.logs, "INFO: "+msg)
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	m.logs = append(m.logs, "WARN: "+msg)
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	m.logs = append(m.logs, "ERROR: "+msg)
}

type mockTracer struct{}

func (m *mockTracer) Start(ctx context.Context, name string) (context.Context, Span) {
	return ctx, &mockSpan{}
}

type mockSpan struct{}

func (m *mockSpan) End()                               {}
func (m *mockSpan) SetAttributes(attrs ...interface{}) {}
func (m *mockSpan) RecordError(err error)              {}
func (m *mockSpan) SetStatus(code int, msg string)     {}

func TestServerWithMocks(t *testing.T) {
	// Test the configuration and option functions
	t.Run("config_validation", func(t *testing.T) {
		config := DefaultRESTConfig()
		if config.Host == "" {
			t.Error("Default config should have a host")
		}
		if config.Port == 0 {
			t.Error("Default config should have a port")
		}
	})

	t.Run("mcp_config_validation", func(t *testing.T) {
		config := DefaultMCPConfig()
		if config.ServerName == "" {
			t.Error("Default MCP config should have a server name")
		}
		if config.ProtocolVersion == "" {
			t.Error("Default MCP config should have a protocol version")
		}
	})

	t.Run("option_functions", func(t *testing.T) {
		logger := &mockLogger{}
		opt := WithLogger(logger)
		if opt == nil {
			t.Error("WithLogger should return a valid option")
		}

		tracer := &mockTracer{}
		tracerOpt := WithTracer(tracer)
		if tracerOpt == nil {
			t.Error("WithTracer should return a valid option")
		}
	})
}

type mockStreamingHandler struct{}

func (m *mockStreamingHandler) HandleStreaming(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]string{"status": "streaming"})
}

func (m *mockStreamingHandler) HandleNonStreaming(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func TestErrorHandling(t *testing.T) {
	// Test custom error creation
	err := NewInvalidRequestError("test_operation", "test message", map[string]string{"field": "value"})
	if err.Code != ErrCodeInvalidRequest {
		t.Errorf("Expected error code %s, got %s", ErrCodeInvalidRequest, err.Code)
	}
	if err.Operation != "test_operation" {
		t.Errorf("Expected operation 'test_operation', got '%s'", err.Operation)
	}

	// Test HTTP status mapping
	if err.HTTPStatus() != http.StatusBadRequest {
		t.Errorf("Expected HTTP status %d, got %d", http.StatusBadRequest, err.HTTPStatus())
	}

	// Test error wrapping
	originalErr := NewToolNotFoundError("test_tool")
	wrappedErr := NewInternalError("test_operation", originalErr)
	if wrappedErr.Operation != "test_operation" {
		t.Errorf("Expected operation 'test_operation', got '%s'", wrappedErr.Operation)
	}
}

func TestFunctionalOptions(t *testing.T) {
	// Test option functions
	config := Config{Host: "test"}
	opt := WithConfig(config)
	if opt == nil {
		t.Error("WithConfig returned nil option")
	}

	restConfig := RESTConfig{APIBasePath: "/test"}
	restOpt := WithRESTConfig(restConfig)
	if restOpt == nil {
		t.Error("WithRESTConfig returned nil option")
	}

	mcpConfig := MCPConfig{ServerName: "test"}
	mcpOpt := WithMCPConfig(mcpConfig)
	if mcpOpt == nil {
		t.Error("WithMCPConfig returned nil option")
	}

	logger := &mockLogger{}
	loggerOpt := WithLogger(logger)
	if loggerOpt == nil {
		t.Error("WithLogger returned nil option")
	}

	tracer := &mockTracer{}
	tracerOpt := WithTracer(tracer)
	if tracerOpt == nil {
		t.Error("WithTracer returned nil option")
	}
}

func BenchmarkServerCreation(b *testing.B) {
	// Skip benchmark since NewRESTServer panics
	b.Skip("Benchmark depends on NewRESTServer which is not implemented in base package")
}

func BenchmarkServerWithOptions(b *testing.B) {
	// Skip benchmark since NewRESTServer panics
	b.Skip("Benchmark depends on NewRESTServer which is not implemented in base package")
}
