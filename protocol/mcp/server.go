package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/protocol/internal/httpsafe"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// Protocol versions understood by the native server and client.
const (
	// ProtocolVersionLatest is the MCP revision the native client requests.
	ProtocolVersionLatest = "2025-06-18"
	// protocolVersionLegacy is the revision the server answers with when the
	// client does not request a supported one (pre-existing behaviour).
	protocolVersionLegacy = "2025-03-26"
)

// MCPServer exposes Beluga tools via the MCP protocol using the Streamable
// HTTP transport. It processes JSON-RPC 2.0 requests at a single HTTP
// endpoint.
//
// Supported methods: initialize, ping, tools/list, tools/call, and the
// list-only resources/list and prompts/list. Because Resource and Prompt
// carry metadata only (no contents or templates), resources/read and
// prompts/get are not implemented and the Resources/Prompts capabilities are
// not advertised. Notifications (messages without an id) are acknowledged
// with HTTP 202 and no body.
//
// Request bodies are limited to 1 MiB. Internal tool errors are logged
// server-side and returned to clients only as a generic "internal error"
// with a request_id in the error data.
//
// Origin validation (DNS-rebinding protection) is opt-in via
// WithAllowedOrigins; by default no Origin check is performed, preserving
// backwards compatibility. Servers reachable from browsers, or bound to
// localhost, should configure it. The server performs no authentication;
// wrap Handler with auth middleware. For a full spec implementation use
// protocol/mcp/sdk.
type MCPServer struct { //nolint:revive // exported name is public API; renaming to remove the stutter would break callers
	name           string
	version        string
	tools          []tool.Tool
	resources      []Resource
	prompts        []Prompt
	capabilities   ServerCapabilities
	allowedOrigins map[string]struct{}
	maxBodyBytes   int64
	mu             sync.RWMutex
}

// ServerOption configures an MCPServer.
type ServerOption func(*MCPServer)

// WithAllowedOrigins enables Origin header validation: a request carrying an
// Origin header that is not in origins (compared case-insensitively, e.g.
// "https://app.example.com") is rejected with HTTP 403. Requests without an
// Origin header (non-browser clients) are allowed. Calling it with no
// origins rejects every request that carries an Origin header.
func WithAllowedOrigins(origins ...string) ServerOption {
	return func(s *MCPServer) {
		s.allowedOrigins = make(map[string]struct{}, len(origins))
		for _, o := range origins {
			s.allowedOrigins[strings.ToLower(strings.TrimRight(o, "/"))] = struct{}{}
		}
	}
}

// WithMaxBodyBytes sets the request-body limit. Non-positive values are
// ignored (the 1 MiB default applies).
func WithMaxBodyBytes(n int64) ServerOption {
	return func(s *MCPServer) {
		if n > 0 {
			s.maxBodyBytes = n
		}
	}
}

// NewServer creates a new MCP server with the given name and version.
func NewServer(name, version string, opts ...ServerOption) *MCPServer {
	s := &MCPServer{
		name:    name,
		version: version,
		capabilities: ServerCapabilities{
			Tools: &ToolCapability{},
		},
		maxBodyBytes: httpsafe.DefaultMaxBodyBytes,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// AddTool registers a Beluga tool with the MCP server.
func (s *MCPServer) AddTool(t tool.Tool) *MCPServer {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, t)
	return s
}

// AddResource registers a resource with the MCP server.
func (s *MCPServer) AddResource(r Resource) *MCPServer {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources = append(s.resources, r)
	return s
}

// AddPrompt registers a prompt template with the MCP server.
func (s *MCPServer) AddPrompt(p Prompt) *MCPServer {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prompts = append(s.prompts, p)
	return s
}

// Handler returns an http.Handler that processes MCP JSON-RPC requests.
func (s *MCPServer) Handler() http.Handler {
	return http.HandlerFunc(s.handleRequest)
}

// Serve starts the MCP server on the given address. It blocks until the context
// is canceled or an error occurs.
func (s *MCPServer) Serve(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: httpsafe.DefaultReadHeaderTimeout,
		ReadTimeout:       httpsafe.DefaultReadTimeout,
		IdleTimeout:       httpsafe.DefaultIdleTimeout,
		// WriteTimeout is intentionally 0: tools/call responses are written
		// only after the tool finishes, and tool runtimes are unbounded by
		// design. Bound tool execution via the request context instead.
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return core.Errorf(core.ErrProviderDown, "mcp/serve: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		if shutdownErr := srv.Close(); shutdownErr != nil {
			return core.Errorf(core.ErrProviderDown, "mcp/serve: shutdown: %w", shutdownErr)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return core.Errorf(core.ErrProviderDown, "mcp/serve: %w", err)
	}
}

// originAllowed reports whether r passes the optional Origin allowlist.
func (s *MCPServer) originAllowed(r *http.Request) bool {
	if s.allowedOrigins == nil {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	_, ok := s.allowedOrigins[strings.ToLower(strings.TrimRight(origin, "/"))]
	return ok
}

func (s *MCPServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	if !s.originAllowed(r) {
		writeErrorStatus(w, http.StatusForbidden, nil, CodeInvalidRequest, "origin not allowed")
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeErrorStatus(w, http.StatusMethodNotAllowed, nil, CodeInvalidRequest, "only POST is supported")
		return
	}

	var req Request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, s.maxBodyBytes)).Decode(&req); err != nil {
		if httpsafe.IsBodyTooLarge(err) {
			writeErrorStatus(w, http.StatusRequestEntityTooLarge, nil, CodeInvalidRequest, "request body too large")
			return
		}
		writeErrorStatus(w, http.StatusBadRequest, nil, CodeParseError, "invalid JSON")
		return
	}

	if req.JSONRPC != "2.0" {
		if req.ID == nil {
			writeErrorStatus(w, http.StatusBadRequest, nil, CodeInvalidRequest, "jsonrpc must be \"2.0\"")
			return
		}
		writeError(w, req.ID, CodeInvalidRequest, "jsonrpc must be \"2.0\"")
		return
	}

	// Notifications (and client responses) carry no id: per the Streamable
	// HTTP transport they are acknowledged with 202 Accepted and no body.
	if req.ID == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	switch req.Method {
	case "initialize":
		s.handleInitialize(w, req)
	case "ping":
		writeResult(w, req.ID, struct{}{})
	case "tools/list":
		s.handleToolsList(w, req)
	case "tools/call":
		s.handleToolsCall(r.Context(), w, req)
	case "resources/list":
		s.handleResourcesList(w, req)
	case "prompts/list":
		s.handlePromptsList(w, req)
	default:
		writeError(w, req.ID, CodeMethodNotFound, "method not found")
	}
}

func (s *MCPServer) handleInitialize(w http.ResponseWriter, req Request) {
	version := protocolVersionLegacy
	if params, ok := req.Params.(map[string]any); ok {
		if v, _ := params["protocolVersion"].(string); v == ProtocolVersionLatest || v == protocolVersionLegacy {
			version = v
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := InitializeResult{
		ProtocolVersion: version,
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
		Capabilities: s.capabilities,
	}

	writeResult(w, req.ID, result)
}

func (s *MCPServer) handleToolsList(w http.ResponseWriter, req Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]ToolInfo, len(s.tools))
	for i, t := range s.tools {
		tools[i] = ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		}
	}

	writeResult(w, req.ID, map[string]any{"tools": tools})
}

func (s *MCPServer) handleToolsCall(ctx context.Context, w http.ResponseWriter, req Request) {
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		writeError(w, req.ID, CodeInvalidParams, "invalid params")
		return
	}

	var params ToolCallParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		writeError(w, req.ID, CodeInvalidParams, "invalid params")
		return
	}

	// Copy the tool out under the lock and release it before executing, so
	// long-running tools do not block AddTool and other writers.
	var target tool.Tool
	s.mu.RLock()
	for _, t := range s.tools {
		if t.Name() == params.Name {
			target = t
			break
		}
	}
	s.mu.RUnlock()

	if target == nil {
		writeError(w, req.ID, CodeInvalidParams, "unknown tool")
		return
	}

	result, err := target.Execute(ctx, params.Arguments)
	if err != nil {
		reqID := httpsafe.NewRequestID()
		slog.Error("mcp: tool execution failed",
			"request_id", reqID,
			"tool", httpsafe.LogValue(params.Name),
			"err", httpsafe.LogValue(err.Error()))
		writeRPCError(w, http.StatusOK, req.ID, &RPCError{
			Code:    CodeInternalError,
			Message: httpsafe.GenericInternalError,
			Data:    map[string]string{"request_id": reqID},
		})
		return
	}

	content := make([]ContentItem, 0, len(result.Content))
	for _, part := range result.Content {
		if tp, ok := part.(schema.TextPart); ok {
			content = append(content, ContentItem{Type: "text", Text: tp.Text})
		}
	}

	writeResult(w, req.ID, ToolCallResult{
		Content: content,
		IsError: result.IsError,
	})
}

func (s *MCPServer) handleResourcesList(w http.ResponseWriter, req Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := make([]Resource, len(s.resources))
	copy(resources, s.resources)

	writeResult(w, req.ID, map[string]any{"resources": resources})
}

func (s *MCPServer) handlePromptsList(w http.ResponseWriter, req Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prompts := make([]Prompt, len(s.prompts))
	copy(prompts, s.prompts)

	writeResult(w, req.ID, map[string]any{"prompts": prompts})
}

func writeResult(w http.ResponseWriter, id any, result any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, id any, code int, message string) {
	writeRPCError(w, http.StatusOK, id, &RPCError{Code: code, Message: message})
}

// writeErrorStatus writes a JSON-RPC error body with an explicit HTTP status.
func writeErrorStatus(w http.ResponseWriter, status int, id any, code int, message string) {
	writeRPCError(w, status, id, &RPCError{Code: code, Message: message})
}

func writeRPCError(w http.ResponseWriter, status int, id any, rpcErr *RPCError) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   rpcErr,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
