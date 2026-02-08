package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// MCPServer exposes Beluga tools, resources, and prompts via the MCP protocol
// using Streamable HTTP transport. It processes JSON-RPC 2.0 requests at a
// single HTTP endpoint.
type MCPServer struct {
	name         string
	version      string
	tools        []tool.Tool
	resources    []Resource
	prompts      []Prompt
	capabilities ServerCapabilities
	mu           sync.RWMutex
}

// NewServer creates a new MCP server with the given name and version.
func NewServer(name, version string) *MCPServer {
	return &MCPServer{
		name:    name,
		version: version,
		capabilities: ServerCapabilities{
			Tools:     &ToolCapability{},
			Resources: &ResourceCapability{},
			Prompts:   &PromptCapability{},
		},
	}
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
		Addr:    addr,
		Handler: s.Handler(),
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("mcp/serve: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		if shutdownErr := srv.Close(); shutdownErr != nil {
			return fmt.Errorf("mcp/serve: shutdown: %w", shutdownErr)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("mcp/serve: %w", err)
	}
}

func (s *MCPServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, nil, CodeInvalidRequest, "only POST is supported")
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nil, CodeParseError, "invalid JSON: "+err.Error())
		return
	}

	if req.JSONRPC != "2.0" {
		writeError(w, req.ID, CodeInvalidRequest, "jsonrpc must be \"2.0\"")
		return
	}

	switch req.Method {
	case "initialize":
		s.handleInitialize(w, req)
	case "tools/list":
		s.handleToolsList(w, req)
	case "tools/call":
		s.handleToolsCall(r.Context(), w, req)
	case "resources/list":
		s.handleResourcesList(w, req)
	case "prompts/list":
		s.handlePromptsList(w, req)
	default:
		writeError(w, req.ID, CodeMethodNotFound, "unknown method: "+req.Method)
	}
}

func (s *MCPServer) handleInitialize(w http.ResponseWriter, req Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := InitializeResult{
		ProtocolVersion: "2025-03-26",
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		writeError(w, req.ID, CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	var params ToolCallParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		writeError(w, req.ID, CodeInvalidParams, "invalid params: "+err.Error())
		return
	}

	var target tool.Tool
	for _, t := range s.tools {
		if t.Name() == params.Name {
			target = t
			break
		}
	}

	if target == nil {
		writeError(w, req.ID, CodeInvalidParams, "unknown tool: "+params.Name)
		return
	}

	result, err := target.Execute(ctx, params.Arguments)
	if err != nil {
		writeError(w, req.ID, CodeInternalError, "tool execution failed: "+err.Error())
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
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, id any, code int, message string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
