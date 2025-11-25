// Package server provides comprehensive tests for the server package.
// These tests are designed to support both unit testing and integration testing.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/server/iface"
)

// Enhanced Mock Implementations

type mockLogger struct {
	logs []logEntry
	mu   sync.RWMutex
}

type logEntry struct {
	level   string
	message string
	args    []interface{}
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		logs: make([]logEntry, 0),
	}
}

func (m *mockLogger) Debug(msg string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logEntry{level: "DEBUG", message: msg, args: args})
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logEntry{level: "INFO", message: msg, args: args})
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logEntry{level: "WARN", message: msg, args: args})
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logEntry{level: "ERROR", message: msg, args: args})
}

func (m *mockLogger) getLogs(level string) []logEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var filtered []logEntry
	for _, log := range m.logs {
		if log.level == level {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

func (m *mockLogger) hasLog(level, message string) bool {
	logs := m.getLogs(level)
	for _, log := range logs {
		if strings.Contains(log.message, message) {
			return true
		}
	}
	return false
}

type mockTracer struct {
	spans []*mockSpan
	mu    sync.RWMutex
}

type mockSpan struct {
	name       string
	attributes map[string]interface{}
	ended      bool
}

func newMockTracer() *mockTracer {
	return &mockTracer{
		spans: make([]*mockSpan, 0),
	}
}

func (m *mockTracer) Start(ctx context.Context, name string) (context.Context, Span) {
	m.mu.Lock()
	defer m.mu.Unlock()
	span := &mockSpan{
		name:       name,
		attributes: make(map[string]interface{}),
		ended:      false,
	}
	m.spans = append(m.spans, span)
	return ctx, span
}

func (m *mockSpan) End() {
	// Mark span as ended
}

func (m *mockSpan) SetAttributes(attrs ...interface{}) {
	// Store attributes for testing
	for i := 0; i < len(attrs); i += 2 {
		if i+1 < len(attrs) {
			// attrs[i] is key, attrs[i+1] is value
		}
	}
}

func (m *mockSpan) RecordError(err error) {
	// Record error for testing
}

func (m *mockSpan) SetStatus(code int, msg string) {
	// Set status for testing
}

func (m *mockTracer) getSpans(name string) []*mockSpan {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var filtered []*mockSpan
	for _, span := range m.spans {
		if span.name == name {
			filtered = append(filtered, span)
		}
	}
	return filtered
}

// Simplified mock meter for testing - avoids complex interface matching
type mockMeter struct{}

func newMockMeter() *mockMeter {
	return &mockMeter{}
}

type mockStreamingHandler struct {
	shouldError       bool
	streamingCalls    int
	nonStreamingCalls int
}

func newMockStreamingHandler() *mockStreamingHandler {
	return &mockStreamingHandler{}
}

func (m *mockStreamingHandler) HandleStreaming(w http.ResponseWriter, r *http.Request) error {
	m.streamingCalls++
	if m.shouldError {
		return fmt.Errorf("streaming handler error")
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "streaming",
		"handler": "mock",
		"calls":   m.streamingCalls,
	})
}

func (m *mockStreamingHandler) HandleNonStreaming(w http.ResponseWriter, r *http.Request) error {
	m.nonStreamingCalls++
	if m.shouldError {
		return fmt.Errorf("non-streaming handler error")
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"handler": "mock",
		"calls":   m.nonStreamingCalls,
	})
}

type mockMCPTool struct {
	name        string
	description string
	shouldError bool
	callCount   int
}

func newMockMCPTool(name, description string) *mockMCPTool {
	return &mockMCPTool{
		name:        name,
		description: description,
		callCount:   0,
	}
}

func (m *mockMCPTool) Name() string {
	return m.name
}

func (m *mockMCPTool) Description() string {
	return m.description
}

func (m *mockMCPTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type": "string",
			},
		},
	}
}

func (m *mockMCPTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	m.callCount++
	if m.shouldError {
		return nil, fmt.Errorf("tool execution error")
	}
	return map[string]interface{}{
		"tool":   m.name,
		"input":  input,
		"calls":  m.callCount,
		"result": "success",
	}, nil
}

type mockMCPResource struct {
	uri         string
	name        string
	description string
	mimeType    string
	content     string
	shouldError bool
	readCount   int
}

func newMockMCPResource(uri, name, description, mimeType, content string) *mockMCPResource {
	return &mockMCPResource{
		uri:         uri,
		name:        name,
		description: description,
		mimeType:    mimeType,
		content:     content,
		readCount:   0,
	}
}

func (m *mockMCPResource) URI() string {
	return m.uri
}

func (m *mockMCPResource) Name() string {
	return m.name
}

func (m *mockMCPResource) Description() string {
	return m.description
}

func (m *mockMCPResource) MimeType() string {
	return m.mimeType
}

func (m *mockMCPResource) Read(ctx context.Context) ([]byte, error) {
	m.readCount++
	if m.shouldError {
		return nil, fmt.Errorf("resource read error")
	}
	return []byte(m.content), nil
}

// Test Suites

func TestNewRESTServer(t *testing.T) {
	tests := []struct {
		name    string
		opts    []iface.Option
		wantErr bool
	}{
		{
			name:    "default_config",
			opts:    nil,
			wantErr: false,
		},
		{
			name: "with_custom_config",
			opts: []iface.Option{
				WithRESTConfig(RESTConfig{
					Config:      Config{Host: "127.0.0.1", Port: 9090},
					APIBasePath: "/api/v2",
				}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewRESTServer(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRESTServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && server == nil {
				t.Error("NewRESTServer() returned nil server")
			}
			if server != nil {
				// Test that it implements the RESTServer interface
				var _ RESTServer = server
			}
		})
	}
}

func TestNewMCPServer(t *testing.T) {
	tests := []struct {
		name    string
		opts    []iface.Option
		wantErr bool
	}{
		{
			name:    "default_config",
			opts:    nil,
			wantErr: false,
		},
		{
			name: "with_custom_config",
			opts: []iface.Option{
				WithMCPConfig(MCPConfig{
					Config:     Config{Host: "127.0.0.1", Port: 9091},
					ServerName: "test-server",
				}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewMCPServer(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMCPServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && server == nil {
				t.Error("NewMCPServer() returned nil server")
			}
			if server != nil {
				// Test that it implements the MCPServer interface
				var _ MCPServer = server
			}
		})
	}
}

func TestDefaultConfigs(t *testing.T) {
	t.Run("rest_config", func(t *testing.T) {
		config := DefaultRESTConfig()

		// Test base config values
		if config.Host != "localhost" {
			t.Errorf("DefaultRESTConfig().Host = %v, want localhost", config.Host)
		}
		if config.Port != 8080 {
			t.Errorf("DefaultRESTConfig().Port = %v, want 8080", config.Port)
		}
		if config.APIBasePath != "/api/v1" {
			t.Errorf("DefaultRESTConfig().APIBasePath = %v, want /api/v1", config.APIBasePath)
		}
		if !config.EnableStreaming {
			t.Error("DefaultRESTConfig().EnableStreaming should be true")
		}
		if !config.EnableCORS {
			t.Error("DefaultRESTConfig().EnableCORS should be true")
		}
	})

	t.Run("mcp_config", func(t *testing.T) {
		config := DefaultMCPConfig()

		// Test base config values
		if config.Host != "localhost" {
			t.Errorf("DefaultMCPConfig().Host = %v, want localhost", config.Host)
		}
		if config.Port != 8081 {
			t.Errorf("DefaultMCPConfig().Port = %v, want 8081", config.Port)
		}
		if config.ServerName != "beluga-mcp-server" {
			t.Errorf("DefaultMCPConfig().ServerName = %v, want beluga-mcp-server", config.ServerName)
		}
		if config.ProtocolVersion != "2024-11-05" {
			t.Errorf("DefaultMCPConfig().ProtocolVersion = %v, want 2024-11-05", config.ProtocolVersion)
		}
	})
}

func TestMiddlewareFunctions(t *testing.T) {
	t.Run("cors_middleware", func(t *testing.T) {
		tests := []struct {
			name           string
			allowedOrigins []string
			requestOrigin  string
			expectHeader   bool
		}{
			{
				name:           "allowed_origin",
				allowedOrigins: []string{"http://example.com"},
				requestOrigin:  "http://example.com",
				expectHeader:   true,
			},
			{
				name:           "wildcard_origin",
				allowedOrigins: []string{"*"},
				requestOrigin:  "http://example.com",
				expectHeader:   true,
			},
			{
				name:           "disallowed_origin",
				allowedOrigins: []string{"http://example.com"},
				requestOrigin:  "http://invalid.com",
				expectHeader:   false,
			},
			{
				name:           "no_origin_header",
				allowedOrigins: []string{"http://example.com"},
				requestOrigin:  "",
				expectHeader:   false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				middleware := CORSMiddleware(tt.allowedOrigins)
				handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))

				req := httptest.NewRequest("GET", "/test", nil)
				if tt.requestOrigin != "" {
					req.Header.Set("Origin", tt.requestOrigin)
				}
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				got := w.Header().Get("Access-Control-Allow-Origin")
				if tt.expectHeader && got == "" {
					t.Error("Expected CORS header to be set")
				}
				if !tt.expectHeader && got != "" {
					t.Errorf("Expected no CORS header, got %s", got)
				}
				if tt.expectHeader && tt.requestOrigin != "" && got != tt.requestOrigin && tt.allowedOrigins[0] != "*" {
					t.Errorf("Expected CORS origin %s, got %s", tt.requestOrigin, got)
				}
			})
		}
	})

	t.Run("cors_preflight", func(t *testing.T) {
		middleware := CORSMiddleware([]string{"http://example.com"})
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for OPTIONS request")
		}))

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS request, got %d", w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected Access-Control-Allow-Methods header")
		}
	})

	t.Run("logging_middleware", func(t *testing.T) {
		logger := newMockLogger()
		middleware := LoggingMiddleware(logger)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("User-Agent", "test-agent")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if !logger.hasLog("INFO", "HTTP Request") {
			t.Error("Expected log entry for HTTP request")
		}
	})

	t.Run("recovery_middleware", func(t *testing.T) {
		logger := newMockLogger()
		middleware := RecoveryMiddleware(logger)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 for panic, got %d", w.Code)
		}
		if !logger.hasLog("ERROR", "Panic recovered") {
			t.Error("Expected log entry for panic recovery")
		}
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("error_creation", func(t *testing.T) {
		tests := []struct {
			name         string
			createError  func() *ServerError
			expectedCode ErrorCode
			expectedOp   string
		}{
			{
				name: "invalid_request",
				createError: func() *ServerError {
					return NewInvalidRequestError("test_op", "test message", map[string]string{"field": "value"})
				},
				expectedCode: ErrCodeInvalidRequest,
				expectedOp:   "test_op",
			},
			{
				name: "not_found",
				createError: func() *ServerError {
					return NewNotFoundError("test_op", "resource")
				},
				expectedCode: ErrCodeNotFound,
				expectedOp:   "test_op",
			},
			{
				name: "internal_error",
				createError: func() *ServerError {
					return NewInternalError("test_op", fmt.Errorf("underlying error"))
				},
				expectedCode: ErrCodeInternalError,
				expectedOp:   "test_op",
			},
			{
				name: "tool_not_found",
				createError: func() *ServerError {
					return NewToolNotFoundError("test_tool")
				},
				expectedCode: ErrCodeToolNotFound,
				expectedOp:   "tool_execution",
			},
			{
				name: "resource_not_found",
				createError: func() *ServerError {
					return NewResourceNotFoundError("test_resource")
				},
				expectedCode: ErrCodeResourceNotFound,
				expectedOp:   "resource_read",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.createError()
				if err.Code != tt.expectedCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedCode, err.Code)
				}
				if err.Operation != tt.expectedOp {
					t.Errorf("Expected operation '%s', got '%s'", tt.expectedOp, err.Operation)
				}
				if err.Message == "" {
					t.Error("Expected non-empty error message")
				}
			})
		}
	})

	t.Run("http_status_mapping", func(t *testing.T) {
		tests := []struct {
			name         string
			createError  func() *ServerError
			expectedCode int
		}{
			{
				name: "invalid_request_400",
				createError: func() *ServerError {
					return NewInvalidRequestError("test", "msg", nil)
				},
				expectedCode: http.StatusBadRequest,
			},
			{
				name: "not_found_404",
				createError: func() *ServerError {
					return NewNotFoundError("test", "resource")
				},
				expectedCode: http.StatusNotFound,
			},
			{
				name: "unauthorized_401",
				createError: func() *ServerError {
					return &ServerError{Code: ErrCodeUnauthorized}
				},
				expectedCode: http.StatusUnauthorized,
			},
			{
				name: "forbidden_403",
				createError: func() *ServerError {
					return &ServerError{Code: ErrCodeForbidden}
				},
				expectedCode: http.StatusForbidden,
			},
			{
				name: "internal_error_500",
				createError: func() *ServerError {
					return NewInternalError("test", nil)
				},
				expectedCode: http.StatusInternalServerError,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.createError()
				if err.HTTPStatus() != tt.expectedCode {
					t.Errorf("Expected HTTP status %d, got %d", tt.expectedCode, err.HTTPStatus())
				}
			})
		}
	})

	t.Run("error_wrapping", func(t *testing.T) {
		originalErr := NewToolNotFoundError("test_tool")
		wrappedErr := NewInternalError("test_operation", originalErr)

		if wrappedErr.Operation != "test_operation" {
			t.Errorf("Expected operation 'test_operation', got '%s'", wrappedErr.Operation)
		}
		if wrappedErr.Err != originalErr {
			t.Error("Expected wrapped error to contain original error")
		}
	})

	t.Run("error_string", func(t *testing.T) {
		err := NewInvalidRequestError("test_op", "test message", nil)
		expected := "test_op: test message"
		if err.Error() != expected {
			t.Errorf("Expected error string '%s', got '%s'", expected, err.Error())
		}

		// Test with underlying error
		wrappedErr := NewInternalError("test_op", fmt.Errorf("underlying"))
		if !strings.Contains(wrappedErr.Error(), "underlying") {
			t.Error("Expected error string to contain underlying error")
		}
	})
}

func TestFunctionalOptions(t *testing.T) {
	t.Run("option_functions", func(t *testing.T) {
		tests := []struct {
			name string
			opt  iface.Option
		}{
			{
				name: "with_config",
				opt:  WithConfig(Config{Host: "test"}),
			},
			{
				name: "with_rest_config",
				opt:  WithRESTConfig(RESTConfig{APIBasePath: "/test"}),
			},
			{
				name: "with_mcp_config",
				opt:  WithMCPConfig(MCPConfig{ServerName: "test"}),
			},
			{
				name: "with_logger",
				opt:  WithLogger(newMockLogger()),
			},
			{
				name: "with_tracer",
				opt:  WithTracer(newMockTracer()),
			},
			{
				name: "with_middleware",
				opt:  WithMiddleware(func(http.Handler) http.Handler { return nil }),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.opt == nil {
					t.Errorf("%s returned nil option", tt.name)
				}
			})
		}
	})
}

// REST Server Tests

func TestRESTServerHandlerRegistration(t *testing.T) {
	logger := newMockLogger()
	server, err := NewRESTServer(
		WithLogger(logger),
		WithRESTConfig(RESTConfig{
			Config:      Config{Host: "localhost", Port: 0},
			APIBasePath: "/api/v1",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create REST server: %v", err)
	}

	if restServer, ok := server.(RESTServer); ok {
		handler := newMockStreamingHandler()

		// Test registering a handler
		restServer.RegisterHandler("test", handler)

		// Verify handler was registered (this would need internal access in real implementation)
		if restServer == nil {
			t.Error("Expected REST server to implement RESTServer interface")
		}
	}
}

func TestRESTServerHTTPEndpoints(t *testing.T) {
	logger := newMockLogger()
	tracer := newMockTracer()

	server, err := NewRESTServer(
		WithLogger(logger),
		WithTracer(tracer),
		WithRESTConfig(RESTConfig{
			Config: Config{
				Host: "localhost",
				Port: 0,
			},
			APIBasePath: "/api/v1",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create REST server: %v", err)
	}

	restServer := server.(RESTServer)

	// Register test handlers
	agentHandler := &mockAgentHandler{}
	chainHandler := &mockChainHandler{}

	restServer.RegisterHTTPHandler("POST", "/api/v1/agents/test/execute", agentHandler.handleExecute)
	restServer.RegisterHTTPHandler("GET", "/api/v1/agents/test/status", agentHandler.handleStatus)
	restServer.RegisterHTTPHandler("POST", "/api/v1/chains/test/execute", chainHandler.handleExecute)
	restServer.RegisterHTTPHandler("GET", "/api/v1/chains/test/status", chainHandler.handleStatus)

	// Note: In a real integration test, you would start the server and make HTTP requests
	// For now, we just verify the handlers can be registered
}

type mockAgentHandler struct{}

func (h *mockAgentHandler) handleExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "executed", "agent": "test"})
}

func (h *mockAgentHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "running", "agent": "test"})
}

type mockChainHandler struct{}

func (h *mockChainHandler) handleExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "executed", "chain": "test"})
}

func (h *mockChainHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "running", "chain": "test"})
}

func TestRESTServerMiddleware(t *testing.T) {
	logger := newMockLogger()
	tracer := newMockTracer()

	server, err := NewRESTServer(
		WithLogger(logger),
		WithTracer(tracer),
		WithRESTConfig(RESTConfig{
			Config:      Config{Host: "localhost", Port: 0},
			APIBasePath: "/api/v1",
		}),
		WithMiddleware(CORSMiddleware([]string{"*"})),
		WithMiddleware(LoggingMiddleware(logger)),
	)
	if err != nil {
		t.Fatalf("Failed to create REST server: %v", err)
	}

	if restServer, ok := server.(RESTServer); ok {
		// Test registering additional middleware
		restServer.RegisterMiddleware(RecoveryMiddleware(logger))

		if restServer == nil {
			t.Error("Expected REST server to implement RESTServer interface")
		}
	}
}

// Integration-style tests that can be used as templates

func TestRESTServerIntegration(t *testing.T) {
	// This test demonstrates how to test a REST server with handlers
	logger := newMockLogger()
	tracer := newMockTracer()

	server, err := NewRESTServer(
		WithLogger(logger),
		WithTracer(tracer),
		WithRESTConfig(RESTConfig{
			Config: Config{
				Host: "localhost",
				Port: 0, // Use random port for testing
			},
			APIBasePath: "/api/v1",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create REST server: %v", err)
	}

	// Register a test handler
	handler := newMockStreamingHandler()
	if restServer, ok := server.(RESTServer); ok {
		restServer.RegisterHandler("test", handler)

		// Test that handler was registered
		// Note: This would require access to internal state or a test-specific interface
		// For now, we just verify the server was created successfully
		if restServer == nil {
			t.Error("Expected REST server to implement RESTServer interface")
		}
	}
}

func TestMCPServerIntegration(t *testing.T) {
	// This test demonstrates how to test an MCP server with tools and resources
	logger := newMockLogger()
	tracer := newMockTracer()

	server, err := NewMCPServer(
		WithLogger(logger),
		WithTracer(tracer),
		WithMCPConfig(MCPConfig{
			Config: Config{
				Host: "localhost",
				Port: 0, // Use random port for testing
			},
			ServerName: "test-mcp-server",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}

	// Register test tools and resources
	if mcpServer, ok := server.(MCPServer); ok {
		tool := newMockMCPTool("test-tool", "A test tool")
		err := mcpServer.RegisterTool(tool)
		if err != nil {
			t.Errorf("Failed to register tool: %v", err)
		}

		resource := newMockMCPResource("test://resource", "test-resource", "A test resource", "text/plain", "test content")
		err = mcpServer.RegisterResource(resource)
		if err != nil {
			t.Errorf("Failed to register resource: %v", err)
		}

		// Test tool listing
		tools, err := mcpServer.ListTools(context.Background())
		if err != nil {
			t.Errorf("Failed to list tools: %v", err)
		}
		if len(tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(tools))
		}

		// Test resource listing
		resources, err := mcpServer.ListResources(context.Background())
		if err != nil {
			t.Errorf("Failed to list resources: %v", err)
		}
		if len(resources) != 1 {
			t.Errorf("Expected 1 resource, got %d", len(resources))
		}
	}
}

// Benchmark tests

func BenchmarkNewRESTServer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		server, err := NewRESTServer()
		if err != nil {
			b.Fatal(err)
		}
		_ = server
	}
}

func BenchmarkNewMCPServer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		server, err := NewMCPServer()
		if err != nil {
			b.Fatal(err)
		}
		_ = server
	}
}

func BenchmarkMiddlewareChain(b *testing.B) {
	logger := newMockLogger()
	corsMiddleware := CORSMiddleware([]string{"*"})
	loggingMiddleware := LoggingMiddleware(logger)

	handler := corsMiddleware(loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkErrorCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := NewInvalidRequestError("test", "message", nil)
		_ = err
	}
}
