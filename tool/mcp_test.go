package tool

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestServer creates an httptest.Server that dispatches JSON-RPC methods to
// the provided handler map. Each handler receives the raw params and returns
// (result, rpcError). If method is not found it returns a -32601 error.
func newTestServer(t *testing.T, handlers map[string]func(json.RawMessage) (any, *rpcError), opts ...func(http.Header)) *httptest.Server {
	t.Helper()

	var sessionID string
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}

		var req jsonrpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		for _, fn := range opts {
			fn(w.Header())
		}

		applySessionHeader(w, &sessionID, req.Method)
		dispatchRPC(w, r, req, handlers)
	}))
}

// applySessionHeader manages the test session ID and sets the Mcp-Session-Id
// response header when appropriate.
func applySessionHeader(w http.ResponseWriter, sessionID *string, method string) {
	if *sessionID != "" {
		w.Header().Set("Mcp-Session-Id", *sessionID)
	}
	if method == "initialize" {
		*sessionID = "test-session-id"
		w.Header().Set("Mcp-Session-Id", *sessionID)
	}
}

// dispatchRPC routes a JSON-RPC request to the appropriate handler and writes
// the JSON response. Notifications (ID==0) are acknowledged with an empty result.
func dispatchRPC(w http.ResponseWriter, _ *http.Request, req jsonrpcRequest, handlers map[string]func(json.RawMessage) (any, *rpcError)) {
	w.Header().Set("Content-Type", "application/json")

	if req.ID == 0 {
		json.NewEncoder(w).Encode(jsonrpcResponse{JSONRPC: "2.0"}) //nolint:errcheck
		return
	}

	handler, ok := handlers[req.Method]
	if !ok {
		json.NewEncoder(w).Encode(jsonrpcResponse{ //nolint:errcheck
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: "method not found"},
		})
		return
	}

	raw, _ := json.Marshal(req.Params)
	result, rpcErr := handler(raw)

	resp := jsonrpcResponse{JSONRPC: "2.0", ID: req.ID, Error: rpcErr}
	if result != nil {
		b, _ := json.Marshal(result)
		resp.Result = b
	}
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// defaultHandlers provides initialize and tools/list handlers suitable for
// most tests.
func defaultHandlers() map[string]func(json.RawMessage) (any, *rpcError) {
	return map[string]func(json.RawMessage) (any, *rpcError){
		"initialize": func(_ json.RawMessage) (any, *rpcError) {
			return initializeResult{
				ProtocolVersion: "2025-03-26",
				ServerInfo:      serverInfo{Name: "test-server", Version: "1.0"},
			}, nil
		},
		"tools/list": func(_ json.RawMessage) (any, *rpcError) {
			return toolsListResult{
				Tools: []toolInfo{
					{
						Name:        "echo",
						Description: "echoes input",
						InputSchema: map[string]any{"type": "object"},
					},
					{
						Name:        "add",
						Description: "adds numbers",
						InputSchema: map[string]any{"type": "object"},
					},
				},
			}, nil
		},
		"tools/call": func(raw json.RawMessage) (any, *rpcError) {
			var params toolCallParams
			json.Unmarshal(raw, &params)
			return toolCallResult{
				Content: []contentItem{
					{Type: "text", Text: "hello from " + params.Name},
				},
			}, nil
		},
	}
}

func connectClient(t *testing.T, serverURL string, opts ...MCPOption) *MCPClient {
	t.Helper()
	c := NewMCPClient(serverURL, opts...)
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return c
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func TestNewMCPClient_Defaults(t *testing.T) {
	c := NewMCPClient("http://localhost:8080")
	if c.serverURL != "http://localhost:8080" {
		t.Errorf("serverURL = %q, want %q", c.serverURL, "http://localhost:8080")
	}
	if c.opts.sessionID != "" {
		t.Errorf("sessionID = %q, want empty", c.opts.sessionID)
	}
	if c.opts.headers == nil {
		t.Error("headers should be initialized (non-nil)")
	}
	if c.httpClient != defaultMCPHTTPClient {
		t.Error("httpClient should default to defaultMCPHTTPClient")
	}
}

func TestNewMCPClient_WithOptions(t *testing.T) {
	custom := &http.Client{}
	c := NewMCPClient("http://localhost:8080",
		WithSessionID("sess-123"),
		WithMCPHeaders(map[string]string{"X-Key": "val"}),
		WithHTTPClient(custom),
	)
	if c.opts.sessionID != "sess-123" {
		t.Errorf("sessionID = %q, want %q", c.opts.sessionID, "sess-123")
	}
	if c.opts.headers["X-Key"] != "val" {
		t.Errorf("X-Key header = %q, want %q", c.opts.headers["X-Key"], "val")
	}
	if c.httpClient != custom {
		t.Error("httpClient should be the custom client")
	}
	if c.sessionID != "sess-123" {
		t.Errorf("initial sessionID = %q, want %q", c.sessionID, "sess-123")
	}
}

// ---------------------------------------------------------------------------
// Connect tests
// ---------------------------------------------------------------------------

func TestConnect_Success(t *testing.T) {
	srv := newTestServer(t, defaultHandlers())
	defer srv.Close()

	c := NewMCPClient(srv.URL)
	err := c.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if !c.connected {
		t.Error("expected connected = true")
	}
	if c.sessionID != "test-session-id" {
		t.Errorf("sessionID = %q, want %q", c.sessionID, "test-session-id")
	}
	if c.capabilities == nil {
		t.Error("capabilities should be set")
	}
}

func TestConnect_AlreadyConnected(t *testing.T) {
	srv := newTestServer(t, defaultHandlers())
	defer srv.Close()

	c := connectClient(t, srv.URL)
	err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error for double Connect")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected core.Error, got %T", err)
	}
	if coreErr.Code != core.ErrInvalidInput {
		t.Errorf("code = %q, want %q", coreErr.Code, core.ErrInvalidInput)
	}
}

func TestConnect_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewMCPClient(srv.URL)
	err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected core.Error, got %T", err)
	}
	if coreErr.Code != core.ErrProviderDown {
		t.Errorf("code = %q, want %q", coreErr.Code, core.ErrProviderDown)
	}
	if !core.IsRetryable(err) {
		t.Error("5xx should be retryable")
	}
}

// ---------------------------------------------------------------------------
// ListTools tests
// ---------------------------------------------------------------------------

func TestListTools_NotConnected(t *testing.T) {
	c := NewMCPClient("http://localhost:9999")
	_, err := c.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected core.Error, got %T", err)
	}
	if coreErr.Code != core.ErrInvalidInput {
		t.Errorf("code = %q, want %q", coreErr.Code, core.ErrInvalidInput)
	}
}

func TestListTools_Success(t *testing.T) {
	srv := newTestServer(t, defaultHandlers())
	defer srv.Close()

	c := connectClient(t, srv.URL)
	tools, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2", len(tools))
	}
	if tools[0].Name() != "echo" {
		t.Errorf("tools[0].Name() = %q, want %q", tools[0].Name(), "echo")
	}
	if tools[0].Description() != "echoes input" {
		t.Errorf("tools[0].Description() = %q, want %q", tools[0].Description(), "echoes input")
	}
	if tools[1].Name() != "add" {
		t.Errorf("tools[1].Name() = %q, want %q", tools[1].Name(), "add")
	}
}

func TestListTools_Empty(t *testing.T) {
	handlers := map[string]func(json.RawMessage) (any, *rpcError){
		"initialize": defaultHandlers()["initialize"],
		"tools/list": func(_ json.RawMessage) (any, *rpcError) {
			return toolsListResult{Tools: []toolInfo{}}, nil
		},
	}
	srv := newTestServer(t, handlers)
	defer srv.Close()

	c := connectClient(t, srv.URL)
	tools, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("len(tools) = %d, want 0", len(tools))
	}
}

// ---------------------------------------------------------------------------
// ExecuteTool tests
// ---------------------------------------------------------------------------

func TestExecuteTool_TextResult(t *testing.T) {
	srv := newTestServer(t, defaultHandlers())
	defer srv.Close()

	c := connectClient(t, srv.URL)
	result, err := c.ExecuteTool(context.Background(), "echo", map[string]any{"msg": "hi"})
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	if result.IsError {
		t.Error("expected IsError = false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("len(Content) = %d, want 1", len(result.Content))
	}
	tp, ok := result.Content[0].(schema.TextPart)
	if !ok {
		t.Fatalf("Content[0] type = %T, want schema.TextPart", result.Content[0])
	}
	if tp.Text != "hello from echo" {
		t.Errorf("text = %q, want %q", tp.Text, "hello from echo")
	}
}

func TestExecuteTool_ErrorResult(t *testing.T) {
	handlers := defaultHandlers()
	handlers["tools/call"] = func(_ json.RawMessage) (any, *rpcError) {
		return toolCallResult{
			Content: []contentItem{{Type: "text", Text: "something went wrong"}},
			IsError: true,
		}, nil
	}
	srv := newTestServer(t, handlers)
	defer srv.Close()

	c := connectClient(t, srv.URL)
	result, err := c.ExecuteTool(context.Background(), "fail", nil)
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError = true")
	}
	tp := result.Content[0].(schema.TextPart)
	if tp.Text != "something went wrong" {
		t.Errorf("text = %q, want %q", tp.Text, "something went wrong")
	}
}

func TestExecuteTool_RPCError(t *testing.T) {
	handlers := defaultHandlers()
	handlers["tools/call"] = func(_ json.RawMessage) (any, *rpcError) {
		return nil, &rpcError{Code: -32600, Message: "invalid request"}
	}
	srv := newTestServer(t, handlers)
	defer srv.Close()

	c := connectClient(t, srv.URL)
	_, err := c.ExecuteTool(context.Background(), "bad", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected core.Error, got %T", err)
	}
	if coreErr.Code != core.ErrToolFailed {
		t.Errorf("code = %q, want %q", coreErr.Code, core.ErrToolFailed)
	}
}

func TestExecuteTool_NotConnected(t *testing.T) {
	c := NewMCPClient("http://localhost:9999")
	_, err := c.ExecuteTool(context.Background(), "echo", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected core.Error, got %T", err)
	}
	if coreErr.Code != core.ErrInvalidInput {
		t.Errorf("code = %q, want %q", coreErr.Code, core.ErrInvalidInput)
	}
}

// ---------------------------------------------------------------------------
// Close tests
// ---------------------------------------------------------------------------

func TestClose_Idempotent(t *testing.T) {
	c := NewMCPClient("http://localhost:9999")
	// Close on unconnected client should return nil.
	if err := c.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestClose_SendsDelete(t *testing.T) {
	var gotMethod string
	var gotSessionHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			gotMethod = r.Method
			gotSessionHeader = r.Header.Get("Mcp-Session-Id")
			w.WriteHeader(http.StatusOK)
			return
		}
		// Handle JSON-RPC for Connect.
		var req jsonrpcRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "del-session")
		if req.ID == 0 {
			json.NewEncoder(w).Encode(jsonrpcResponse{JSONRPC: "2.0"})
			return
		}
		result, _ := json.Marshal(initializeResult{
			ProtocolVersion: "2025-03-26",
			ServerInfo:      serverInfo{Name: "test", Version: "1.0"},
		})
		json.NewEncoder(w).Encode(jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		})
	}))
	defer srv.Close()

	c := connectClient(t, srv.URL)
	if err := c.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotSessionHeader != "del-session" {
		t.Errorf("session header = %q, want %q", gotSessionHeader, "del-session")
	}
	if c.connected {
		t.Error("expected connected = false after Close")
	}
}

// ---------------------------------------------------------------------------
// FromMCP tests
// ---------------------------------------------------------------------------

func TestFromMCP_EndToEnd(t *testing.T) {
	srv := newTestServer(t, defaultHandlers())
	defer srv.Close()

	tools, client, err := FromMCP(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FromMCP: %v", err)
	}
	defer client.Close(context.Background())

	if len(tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2", len(tools))
	}

	// Execute a tool via the returned mcpTool.
	result, err := tools[0].Execute(context.Background(), map[string]any{"x": 1})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	tp := result.Content[0].(schema.TextPart)
	if tp.Text != "hello from echo" {
		t.Errorf("text = %q, want %q", tp.Text, "hello from echo")
	}
}

// ---------------------------------------------------------------------------
// Session ID tests
// ---------------------------------------------------------------------------

func TestSessionID_FromServer(t *testing.T) {
	srv := newTestServer(t, defaultHandlers())
	defer srv.Close()

	c := NewMCPClient(srv.URL)
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if c.sessionID != "test-session-id" {
		t.Errorf("sessionID = %q, want %q", c.sessionID, "test-session-id")
	}
}

func TestSessionID_FromOption(t *testing.T) {
	c := NewMCPClient("http://localhost:9999", WithSessionID("preset-id"))
	if c.sessionID != "preset-id" {
		t.Errorf("sessionID = %q, want %q", c.sessionID, "preset-id")
	}
}

// ---------------------------------------------------------------------------
// Custom headers tests
// ---------------------------------------------------------------------------

func TestCustomHeaders_Sent(t *testing.T) {
	var gotAuth string
	var gotCustom string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCustom = r.Header.Get("X-Custom")

		var req jsonrpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		if req.ID == 0 {
			json.NewEncoder(w).Encode(jsonrpcResponse{JSONRPC: "2.0"})
			return
		}
		result, _ := json.Marshal(initializeResult{
			ProtocolVersion: "2025-03-26",
			ServerInfo:      serverInfo{Name: "test", Version: "1.0"},
		})
		json.NewEncoder(w).Encode(jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		})
	}))
	defer srv.Close()

	c := NewMCPClient(srv.URL, WithMCPHeaders(map[string]string{
		"Authorization": "Bearer tok",
		"X-Custom":      "custom-val",
	}))
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer tok")
	}
	if gotCustom != "custom-val" {
		t.Errorf("X-Custom = %q, want %q", gotCustom, "custom-val")
	}
}

// ---------------------------------------------------------------------------
// HTTP error mapping tests
// ---------------------------------------------------------------------------

func TestHTTP_401_MapsToAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewMCPClient(srv.URL)
	err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected core.Error, got %T", err)
	}
	if coreErr.Code != core.ErrAuth {
		t.Errorf("code = %q, want %q", coreErr.Code, core.ErrAuth)
	}
	if core.IsRetryable(err) {
		t.Error("auth error should not be retryable")
	}
}

func TestHTTP_403_MapsToAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := NewMCPClient(srv.URL)
	err := c.Connect(context.Background())
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrAuth {
		t.Errorf("expected ErrAuth, got %v", err)
	}
}

func TestHTTP_429_MapsToRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewMCPClient(srv.URL)
	err := c.Connect(context.Background())
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrRateLimit {
		t.Errorf("expected ErrRateLimit, got %v", err)
	}
	if !core.IsRetryable(err) {
		t.Error("rate limit should be retryable")
	}
}

// ---------------------------------------------------------------------------
// Context cancellation test
// ---------------------------------------------------------------------------

func TestContext_Cancellation(t *testing.T) {
	srv := newTestServer(t, defaultHandlers())
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	c := NewMCPClient(srv.URL)
	err := c.Connect(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// ---------------------------------------------------------------------------
// mcpTool interface compliance
// ---------------------------------------------------------------------------

func TestMCPTool_ImplementsTool(t *testing.T) {
	srv := newTestServer(t, defaultHandlers())
	defer srv.Close()

	c := connectClient(t, srv.URL)
	tools, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	for _, tool := range tools {
		// Verify that each tool implements the Tool interface.
		var _ Tool = tool

		if tool.Name() == "" {
			t.Error("tool name should not be empty")
		}
		if tool.Description() == "" {
			t.Error("tool description should not be empty")
		}
		if tool.InputSchema() == nil {
			t.Error("tool inputSchema should not be nil")
		}
	}
}

// ---------------------------------------------------------------------------
// Option function unit tests
// ---------------------------------------------------------------------------

func TestMCPOption_WithSessionID(t *testing.T) {
	opts := mcpOptions{headers: make(map[string]string)}
	WithSessionID("abc")(&opts)
	if opts.sessionID != "abc" {
		t.Errorf("sessionID = %q, want %q", opts.sessionID, "abc")
	}
}

func TestMCPOption_WithMCPHeaders(t *testing.T) {
	opts := mcpOptions{headers: make(map[string]string)}
	WithMCPHeaders(map[string]string{"key": "value"})(&opts)
	if opts.headers["key"] != "value" {
		t.Errorf("headers[key] = %q, want %q", opts.headers["key"], "value")
	}
}

func TestMCPOption_WithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	opts := mcpOptions{}
	WithHTTPClient(custom)(&opts)
	if opts.httpClient != custom {
		t.Error("httpClient should be the custom client")
	}
}

// ---------------------------------------------------------------------------
// Concurrency safety (used with -race)
// ---------------------------------------------------------------------------

func TestConcurrent_ListAndExecute(t *testing.T) {
	srv := newTestServer(t, defaultHandlers())
	defer srv.Close()

	c := connectClient(t, srv.URL)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = c.ListTools(context.Background())
		}()
		go func() {
			defer wg.Done()
			_, _ = c.ExecuteTool(context.Background(), "echo", map[string]any{"x": 1})
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// URL validation tests (H3)
// ---------------------------------------------------------------------------

func TestConnect_EmptyURL(t *testing.T) {
	c := NewMCPClient("")
	err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestConnect_InvalidScheme(t *testing.T) {
	c := NewMCPClient("ftp://example.com/mcp")
	err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error for ftp scheme")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
	if !strings.Contains(coreErr.Message, "unsupported URL scheme") {
		t.Errorf("expected scheme error, got %q", coreErr.Message)
	}
}

func TestConnect_NoHost(t *testing.T) {
	c := NewMCPClient("http://")
	err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error for URL with no host")
	}
}

// ---------------------------------------------------------------------------
// RPC error message truncation (M1)
// ---------------------------------------------------------------------------

func TestRPCError_MessageTruncated(t *testing.T) {
	longMsg := strings.Repeat("x", 1000)
	handlers := defaultHandlers()
	handlers["tools/call"] = func(_ json.RawMessage) (any, *rpcError) {
		return nil, &rpcError{Code: -32600, Message: longMsg}
	}
	srv := newTestServer(t, handlers)
	defer srv.Close()

	c := connectClient(t, srv.URL)
	_, err := c.ExecuteTool(context.Background(), "bad", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	// The error message should be truncated to maxRPCErrorMessageLen + suffix.
	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected core.Error, got %T", err)
	}
	if len(coreErr.Message) > maxRPCErrorMessageLen+100 {
		t.Errorf("error message too long: %d chars", len(coreErr.Message))
	}
	if !strings.Contains(coreErr.Message, "...(truncated)") {
		t.Error("expected truncation marker in error message")
	}
}

// ---------------------------------------------------------------------------
// InputSchema defensive copy (M3)
// ---------------------------------------------------------------------------

func TestMCPTool_InputSchemaDefensiveCopy(t *testing.T) {
	srv := newTestServer(t, defaultHandlers())
	defer srv.Close()

	c := connectClient(t, srv.URL)
	tools, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	s1 := tools[0].InputSchema()
	s1["mutated"] = true

	s2 := tools[0].InputSchema()
	if _, ok := s2["mutated"]; ok {
		t.Error("InputSchema mutation leaked to internal state")
	}
}

// ---------------------------------------------------------------------------
// Headers defensive copy (M4)
// ---------------------------------------------------------------------------

func TestNewMCPClient_HeadersDefensiveCopy(t *testing.T) {
	original := map[string]string{"X-Key": "val"}
	c := NewMCPClient("http://localhost:8080", WithMCPHeaders(original))

	// Mutate the original map after construction.
	original["X-Injected"] = "bad"

	if _, ok := c.opts.headers["X-Injected"]; ok {
		t.Error("header mutation after construction leaked into client")
	}
}

// ---------------------------------------------------------------------------
// Content-Type validation (L4)
// ---------------------------------------------------------------------------

func TestCall_InvalidContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html>not json</html>"))
	}))
	defer srv.Close()

	c := NewMCPClient(srv.URL)
	// Manually set connected to test call() path.
	c.connected = true
	_, err := c.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error for wrong Content-Type")
	}
	if !strings.Contains(err.Error(), "Content-Type") {
		t.Errorf("expected Content-Type error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Notify status code check (L1)
// ---------------------------------------------------------------------------

func TestNotify_ServerError(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// First call is "initialize" (via call()), let it succeed.
		if callCount == 1 {
			w.Header().Set("Content-Type", "application/json")
			result, _ := json.Marshal(initializeResult{
				ProtocolVersion: "2025-03-26",
				ServerInfo:      serverInfo{Name: "test", Version: "1.0"},
			})
			resp := jsonrpcResponse{JSONRPC: "2.0", ID: 1, Result: result}
			json.NewEncoder(w).Encode(resp)
			return
		}
		// Second call is the notification -- return 500.
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewMCPClient(srv.URL)
	err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error from notify 500")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrProviderDown {
		t.Errorf("expected ErrProviderDown, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Compile-time interface assertion test (M2)
// ---------------------------------------------------------------------------

func TestMCPTool_CompileTimeAssertion(t *testing.T) {
	// This test simply verifies the compile-time assertion exists.
	// var _ Tool = (*mcpTool)(nil) is declared in mcp.go.
	var _ Tool = (*mcpTool)(nil)
}
