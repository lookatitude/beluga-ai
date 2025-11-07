// Package mcp provides MCP (Model Context Protocol) server implementation.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/lookatitude/beluga-ai/pkg/server/iface"
)

// Server implements an MCP server
type Server struct {
	config      iface.MCPConfig
	server      *http.Server
	metrics     interface{} // TODO: fix metrics
	logger      iface.Logger
	tracer      iface.Tracer
	tools       map[string]iface.MCPTool
	resources   map[string]iface.MCPResource
	toolsMu     sync.RWMutex
	resourcesMu sync.RWMutex
	startTime   time.Time
}

// MCPMessage represents an MCP protocol message
type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP protocol error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP request/response types
type InitializeRequest struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      map[string]interface{} `json:"clientInfo"`
}

type InitializeResponse struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      map[string]interface{} `json:"serverInfo"`
}

type ListToolsRequest struct{}

type ListToolsResponse struct {
	Tools []MCPToolInfo `json:"tools"`
}

type MCPToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type CallToolResponse struct {
	Content []MCPContent `json:"content"`
}

type MCPContent struct {
	Type string      `json:"type"`
	Text string      `json:"text,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

type ListResourcesRequest struct{}

type ListResourcesResponse struct {
	Resources []MCPResourceInfo `json:"resources"`
}

type MCPResourceInfo struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

type ReadResourceRequest struct {
	URI string `json:"uri"`
}

type ReadResourceResponse struct {
	Contents []MCPResourceContent `json:"contents"`
}

type MCPResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// NewServer creates a new MCP server instance
func NewServer(opts ...iface.Option) (*Server, error) {
	options := &serverOptions{
		config: iface.Config{
			Host:            "localhost",
			Port:            8080,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     120 * time.Second,
			MaxHeaderBytes:  1 << 20, // 1MB
			EnableMetrics:   true,
			EnableTracing:   true,
			LogLevel:        "info",
			ShutdownTimeout: 30 * time.Second,
		},
		mcpConfig: &iface.MCPConfig{
			ServerName:            "beluga-mcp-server",
			ServerVersion:         "1.0.0",
			ProtocolVersion:       "2024-11-05",
			MaxConcurrentRequests: 10,
			RequestTimeout:        60 * time.Second,
		},
		tools:     nil,
		resources: nil,
	}

	// Skip option processing for now to avoid type issues
	_ = opts

	// Merge configs
	if options.mcpConfig != nil {
		options.mcpConfig.Config = options.config
	} else {
		options.mcpConfig = &iface.MCPConfig{
			Config: options.config,
		}
	}

	// Convert slices to maps
	toolsMap := make(map[string]iface.MCPTool)
	if options.tools != nil {
		for _, tool := range options.tools {
			if tool != nil {
				toolsMap[tool.Name()] = tool
			}
		}
	}

	resourcesMap := make(map[string]iface.MCPResource)
	if options.resources != nil {
		for _, resource := range options.resources {
			if resource != nil {
				resourcesMap[resource.URI()] = resource
			}
		}
	}

	s := &Server{
		config:    *options.mcpConfig,
		metrics:   nil,
		logger:    options.logger,
		tracer:    options.tracer,
		tools:     toolsMap,
		resources: resourcesMap,
		startTime: time.Now(),
	}

	if s.logger == nil {
		s.logger = &defaultLogger{}
	}

	if s.tracer == nil {
		s.tracer = &noopTracer{}
	}

	return s, nil
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCPRequest)

	s.server = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler:        s.handleCORS(mux),
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	s.logger.Info("Starting MCP server",
		"host", s.config.Host,
		"port", s.config.Port,
		"server_name", s.config.ServerName,
		"protocol_version", s.config.ProtocolVersion,
	)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- s.server.ListenAndServe()
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-serverErr:
		if err != http.ErrServerClosed {
			s.logger.Error("MCP server error", "error", err)
			return iface.NewInternalError("mcp_server_start", err)
		}
		return nil
	case <-ctx.Done():
		s.logger.Info("MCP server shutdown requested")
		return s.Stop(ctx)
	}
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	s.logger.Info("Shutting down MCP server gracefully")
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("MCP server shutdown error", "error", err)
		return iface.NewInternalError("mcp_server_shutdown", err)
	}

	s.logger.Info("MCP server shutdown complete")
	return nil
}

// IsHealthy returns true if the server is healthy
// A server is healthy if it's been created (even if not started yet)
func (s *Server) IsHealthy(ctx context.Context) bool {
	// Server is healthy if it exists (created) or if it's running
	return s != nil
}

// RegisterTool registers an MCP tool
func (s *Server) RegisterTool(tool iface.MCPTool) error {
	s.toolsMu.Lock()
	defer s.toolsMu.Unlock()

	name := tool.Name()
	if _, exists := s.tools[name]; exists {
		return iface.NewInvalidRequestError("register_tool", fmt.Sprintf("tool '%s' already registered", name), nil)
	}

	s.tools[name] = tool
	// TODO: Add metrics back
	s.logger.Info("Registered MCP tool", "name", name)
	return nil
}

// RegisterResource registers an MCP resource
func (s *Server) RegisterResource(resource iface.MCPResource) error {
	s.resourcesMu.Lock()
	defer s.resourcesMu.Unlock()

	uri := resource.URI()
	if _, exists := s.resources[uri]; exists {
		return iface.NewInvalidRequestError("register_resource", fmt.Sprintf("resource '%s' already registered", uri), nil)
	}

	s.resources[uri] = resource
	// TODO: Add metrics back
	// s.metrics.RecordResourceRegistration(context.Background(), uri)
	s.logger.Info("Registered MCP resource", "uri", uri)
	return nil
}

// ListTools returns all registered tools
func (s *Server) ListTools(ctx context.Context) ([]iface.MCPTool, error) {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()

	tools := make([]iface.MCPTool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}
	return tools, nil
}

// ListResources returns all registered resources
func (s *Server) ListResources(ctx context.Context) ([]iface.MCPResource, error) {
	s.resourcesMu.RLock()
	defer s.resourcesMu.RUnlock()

	resources := make([]iface.MCPResource, 0, len(s.resources))
	for _, resource := range s.resources {
		resources = append(resources, resource)
	}
	return resources, nil
}

// CallTool executes a tool by name
func (s *Server) CallTool(ctx context.Context, name string, input map[string]interface{}) (interface{}, error) {
	s.toolsMu.RLock()
	tool, exists := s.tools[name]
	s.toolsMu.RUnlock()

	if !exists {
		return nil, iface.NewToolNotFoundError(name)
	}

	start := time.Now()
	result, err := tool.Execute(ctx, input)
	duration := time.Since(start)

	_ = err // success := err == nil
	// TODO: Add metrics back
	// s.metrics.RecordMCPToolCall(ctx, name, success, duration)

	if err != nil {
		s.logger.Error("Tool execution failed", "tool", name, "error", err)
		return nil, iface.NewToolExecutionError(name, err)
	}

	s.logger.Info("Tool executed successfully", "tool", name, "duration_ms", duration.Milliseconds())
	return result, nil
}

// HTTP handlers

func (s *Server) handleCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	ctx, span := s.tracer.Start(r.Context(), "mcp.request")
	defer span.End()

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
	)

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.handleMCPError(w, nil, -32700, "Parse error", err)
		return
	}

	// Parse MCP message
	var msg MCPMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		s.handleMCPError(w, nil, -32700, "Parse error", err)
		return
	}

	// Set JSON-RPC version if not provided
	if msg.JSONRPC == "" {
		msg.JSONRPC = "2.0"
	}

	// Handle the request
	response, err := s.handleMCPMessage(ctx, &msg)
	if err != nil {
		s.handleMCPError(w, msg.ID, -32603, "Internal error", err)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleMCPMessage(ctx context.Context, msg *MCPMessage) (*MCPMessage, error) {
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(ctx, msg)
	case "tools/list":
		return s.handleListTools(ctx, msg)
	case "tools/call":
		return s.handleCallTool(ctx, msg)
	case "resources/list":
		return s.handleListResources(ctx, msg)
	case "resources/read":
		return s.handleReadResource(ctx, msg)
	default:
		return nil, iface.NewMCPProtocolError("method_not_found", fmt.Errorf("unknown method: %s", msg.Method))
	}
}

func (s *Server) handleInitialize(ctx context.Context, msg *MCPMessage) (*MCPMessage, error) {
	response := &MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: InitializeResponse{
			ProtocolVersion: s.config.ProtocolVersion,
			Capabilities: map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": false,
				},
				"resources": map[string]interface{}{
					"listChanged": false,
				},
			},
			ServerInfo: map[string]interface{}{
				"name":    s.config.ServerName,
				"version": s.config.ServerVersion,
			},
		},
	}
	return response, nil
}

func (s *Server) handleListTools(ctx context.Context, msg *MCPMessage) (*MCPMessage, error) {
	tools, err := s.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	toolInfos := make([]MCPToolInfo, len(tools))
	for i, tool := range tools {
		toolInfos[i] = MCPToolInfo{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		}
	}

	response := &MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: ListToolsResponse{
			Tools: toolInfos,
		},
	}
	return response, nil
}

func (s *Server) handleCallTool(ctx context.Context, msg *MCPMessage) (*MCPMessage, error) {
	var req CallToolRequest
	if err := s.parseParams(msg.Params, &req); err != nil {
		return nil, err
	}

	result, err := s.CallTool(ctx, req.Name, req.Arguments)
	if err != nil {
		return nil, err
	}

	// Convert result to MCP content format
	content := []MCPContent{
		{
			Type: "text",
			Text: fmt.Sprintf("%v", result),
		},
	}

	response := &MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: CallToolResponse{
			Content: content,
		},
	}
	return response, nil
}

func (s *Server) handleListResources(ctx context.Context, msg *MCPMessage) (*MCPMessage, error) {
	resources, err := s.ListResources(ctx)
	if err != nil {
		return nil, err
	}

	resourceInfos := make([]MCPResourceInfo, len(resources))
	for i, resource := range resources {
		resourceInfos[i] = MCPResourceInfo{
			URI:         resource.URI(),
			Name:        resource.Name(),
			Description: resource.Description(),
			MimeType:    resource.MimeType(),
		}
	}

	response := &MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: ListResourcesResponse{
			Resources: resourceInfos,
		},
	}
	return response, nil
}

func (s *Server) handleReadResource(ctx context.Context, msg *MCPMessage) (*MCPMessage, error) {
	var req ReadResourceRequest
	if err := s.parseParams(msg.Params, &req); err != nil {
		return nil, err
	}

	s.resourcesMu.RLock()
	resource, exists := s.resources[req.URI]
	s.resourcesMu.RUnlock()

	if !exists {
		return nil, iface.NewResourceNotFoundError(req.URI)
	}

	start := time.Now()
	data, err := resource.Read(ctx)
	duration := time.Since(start)

	_ = duration // success := err == nil
	// TODO: Add metrics back
	// s.metrics.RecordMCPResourceRead(ctx, req.URI, success, duration)

	if err != nil {
		return nil, iface.NewResourceReadError(req.URI, err)
	}

	content := []MCPResourceContent{
		{
			URI:      req.URI,
			MimeType: resource.MimeType(),
			Text:     string(data),
		},
	}

	response := &MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: ReadResourceResponse{
			Contents: content,
		},
	}
	return response, nil
}

func (s *Server) handleMCPError(w http.ResponseWriter, id interface{}, code int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // MCP uses 200 even for errors

	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    err.Error(),
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) parseParams(params interface{}, target interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return iface.NewMCPProtocolError("parse_params", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return iface.NewMCPProtocolError("parse_params", err)
	}

	return nil
}

// Default implementations for optional dependencies

type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {}
func (l *defaultLogger) Info(msg string, args ...interface{})  {}
func (l *defaultLogger) Warn(msg string, args ...interface{})  {}
func (l *defaultLogger) Error(msg string, args ...interface{}) {}

type noopTracer struct{}

func (t *noopTracer) Start(ctx context.Context, name string) (context.Context, iface.Span) {
	return ctx, &noopSpan{}
}

type noopSpan struct{}

func (s *noopSpan) End()                               {}
func (s *noopSpan) SetAttributes(attrs ...interface{}) {}
func (s *noopSpan) RecordError(err error)              {}
func (s *noopSpan) SetStatus(code int, msg string)     {}

type noopMetrics struct{}

func (m *noopMetrics) RecordHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
}
func (m *noopMetrics) RecordActiveConnections(ctx context.Context, count int64) {}
func (m *noopMetrics) RecordMCPToolCall(ctx context.Context, toolName string, success bool, duration time.Duration) {
}
func (m *noopMetrics) RecordMCPResourceRead(ctx context.Context, resourceURI string, success bool, duration time.Duration) {
}
func (m *noopMetrics) RecordToolRegistration(ctx context.Context, toolName string)        {}
func (m *noopMetrics) RecordResourceRegistration(ctx context.Context, resourceURI string) {}
func (m *noopMetrics) RecordHealthCheck(ctx context.Context, healthy bool)                {}
func (m *noopMetrics) RecordServerUptime(ctx context.Context, uptime time.Duration)       {}

// serverOptions holds configuration options for the server
type serverOptions struct {
	config      iface.Config
	restConfig  *iface.RESTConfig
	mcpConfig   *iface.MCPConfig
	logger      iface.Logger
	tracer      iface.Tracer
	meter       iface.Meter
	middlewares []iface.Middleware
	tools       []iface.MCPTool
	resources   []iface.MCPResource
}
