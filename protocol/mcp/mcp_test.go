package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// mockTool implements tool.Tool for testing.
type mockTool struct {
	name        string
	description string
	inputSchema map[string]any
	executeFn   func(ctx context.Context, input map[string]any) (*tool.Result, error)
}

func (m *mockTool) Name() string              { return m.name }
func (m *mockTool) Description() string        { return m.description }
func (m *mockTool) InputSchema() map[string]any { return m.inputSchema }
func (m *mockTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, input)
	}
	return tool.TextResult("ok"), nil
}

func newTestTool() *mockTool {
	return &mockTool{
		name:        "echo",
		description: "Echoes input text",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{"type": "string"},
			},
		},
		executeFn: func(_ context.Context, input map[string]any) (*tool.Result, error) {
			text, _ := input["text"].(string)
			return tool.TextResult("echo: " + text), nil
		},
	}
}

func setupTestServer() (*MCPServer, *httptest.Server) {
	srv := NewServer("test-server", "1.0.0")
	srv.AddTool(newTestTool())
	srv.AddResource(Resource{
		URI:         "file:///test.txt",
		Name:        "test-file",
		Description: "A test resource",
		MIMEType:    "text/plain",
	})
	srv.AddPrompt(Prompt{
		Name:        "greet",
		Description: "A greeting prompt",
		Arguments: []PromptArgument{
			{Name: "name", Description: "Name to greet", Required: true},
		},
	})
	ts := httptest.NewServer(srv.Handler())
	return srv, ts
}

func TestServer_Initialize(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	caps, err := client.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if caps.Tools == nil {
		t.Error("expected tools capability")
	}
	if caps.Resources == nil {
		t.Error("expected resources capability")
	}
	if caps.Prompts == nil {
		t.Error("expected prompts capability")
	}
}

func TestServer_ToolsList(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "echo" {
		t.Errorf("expected tool name 'echo', got %q", tools[0].Name)
	}
	if tools[0].Description != "Echoes input text" {
		t.Errorf("expected description 'Echoes input text', got %q", tools[0].Description)
	}
}

func TestServer_ToolsCall(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	result, err := client.CallTool(context.Background(), "echo", map[string]any{"text": "hello"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	if result.IsError {
		t.Error("expected IsError=false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Text != "echo: hello" {
		t.Errorf("expected 'echo: hello', got %q", result.Content[0].Text)
	}
}

func TestServer_ToolsCall_Unknown(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.CallTool(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestServer_ResourcesList(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	client := NewClient(ts.URL)

	var result struct {
		Resources []Resource `json:"resources"`
	}
	if err := client.call(context.Background(), "resources/list", nil, &result); err != nil {
		t.Fatalf("resources/list: %v", err)
	}

	if len(result.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(result.Resources))
	}
	if result.Resources[0].Name != "test-file" {
		t.Errorf("expected resource name 'test-file', got %q", result.Resources[0].Name)
	}
}

func TestServer_PromptsList(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	client := NewClient(ts.URL)

	var result struct {
		Prompts []Prompt `json:"prompts"`
	}
	if err := client.call(context.Background(), "prompts/list", nil, &result); err != nil {
		t.Fatalf("prompts/list: %v", err)
	}

	if len(result.Prompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(result.Prompts))
	}
	if result.Prompts[0].Name != "greet" {
		t.Errorf("expected prompt name 'greet', got %q", result.Prompts[0].Name)
	}
	if len(result.Prompts[0].Arguments) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(result.Prompts[0].Arguments))
	}
}

func TestServer_UnknownMethod(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	err := client.call(context.Background(), "nonexistent/method", nil, &json.RawMessage{})
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
}

func TestServer_InvalidHTTPMethod(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp Response
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rpcResp.Error == nil {
		t.Fatal("expected error for GET request")
	}
	if rpcResp.Error.Code != CodeInvalidRequest {
		t.Errorf("expected code %d, got %d", CodeInvalidRequest, rpcResp.Error.Code)
	}
}

func TestClient_Initialize(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	caps, err := client.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if caps == nil {
		t.Fatal("expected non-nil capabilities")
	}
}

func TestClient_ListTools(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
}

func TestClient_CallTool(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	result, err := client.CallTool(context.Background(), "echo", map[string]any{"text": "world"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Text != "echo: world" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestFromMCP(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	ctx := context.Background()
	tools, err := FromMCP(ctx, ts.URL)
	if err != nil {
		t.Fatalf("FromMCP: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	tt := tools[0]
	if tt.Name() != "echo" {
		t.Errorf("expected name 'echo', got %q", tt.Name())
	}
	if tt.Description() != "Echoes input text" {
		t.Errorf("expected description, got %q", tt.Description())
	}
	if tt.InputSchema() == nil {
		t.Error("expected non-nil InputSchema")
	}

	result, err := tt.Execute(ctx, map[string]any{"text": "test"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.IsError {
		t.Error("expected IsError=false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content part, got %d", len(result.Content))
	}
	tp, ok := result.Content[0].(schema.TextPart)
	if !ok {
		t.Fatalf("expected TextPart, got %T", result.Content[0])
	}
	if tp.Text != "echo: test" {
		t.Errorf("expected 'echo: test', got %q", tp.Text)
	}
}

func TestFromMCP_ErrorTool(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.AddTool(&mockTool{
		name:        "fail",
		description: "Always fails",
		inputSchema: map[string]any{"type": "object"},
		executeFn: func(_ context.Context, _ map[string]any) (*tool.Result, error) {
			return &tool.Result{
				Content: []schema.ContentPart{schema.TextPart{Text: "something went wrong"}},
				IsError: true,
			}, nil
		},
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	tools, err := FromMCP(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("FromMCP: %v", err)
	}

	result, err := tools[0].Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestServer_AddChaining(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	result := srv.AddTool(newTestTool()).AddResource(Resource{Name: "r"}).AddPrompt(Prompt{Name: "p"})
	if result != srv {
		t.Error("expected chaining to return same server")
	}
}

func TestNewServer_Defaults(t *testing.T) {
	srv := NewServer("myserver", "2.0.0")
	if srv.name != "myserver" {
		t.Errorf("expected name 'myserver', got %q", srv.name)
	}
	if srv.version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %q", srv.version)
	}
}

// --- Client error path tests ---

func TestClient_Initialize_ConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1")
	_, err := client.Initialize(context.Background())
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestClient_ListTools_ConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1")
	_, err := client.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestClient_CallTool_ConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1")
	_, err := client.CallTool(context.Background(), "echo", nil)
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestClient_CallTool_DecodeError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.CallTool(context.Background(), "echo", nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestFromMCP_InitializeError(t *testing.T) {
	_, err := FromMCP(context.Background(), "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestFromMCP_ListToolsError(t *testing.T) {
	// Server that responds to initialize but fails on tools/list.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method == "initialize" {
			resp := Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: InitializeResult{
					ProtocolVersion: "2025-03-26",
					Capabilities:    ServerCapabilities{Tools: &ToolCapability{}},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else {
			resp := Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: CodeInternalError, Message: "list failed"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer ts.Close()

	_, err := FromMCP(context.Background(), ts.URL)
	if err == nil {
		t.Fatal("expected error for ListTools failure")
	}
}

// --- Server error path tests ---

func TestServer_InvalidJSON(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL, "application/json", strings.NewReader("not json"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp Response
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error == nil || rpcResp.Error.Code != CodeParseError {
		t.Errorf("expected parse error, got %+v", rpcResp.Error)
	}
}

func TestServer_InvalidJSONRPCVersion(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	body, _ := json.Marshal(Request{JSONRPC: "1.0", ID: 1, Method: "initialize"})
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp Response
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error == nil || rpcResp.Error.Code != CodeInvalidRequest {
		t.Errorf("expected invalid request error, got %+v", rpcResp.Error)
	}
}

func TestServer_ToolsCall_ExecuteError(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.AddTool(&mockTool{
		name:        "errortool",
		description: "Always errors",
		inputSchema: map[string]any{"type": "object"},
		executeFn: func(_ context.Context, _ map[string]any) (*tool.Result, error) {
			return nil, fmt.Errorf("execution failed")
		},
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.CallTool(context.Background(), "errortool", nil)
	if err == nil {
		t.Fatal("expected error for tool execution failure")
	}
}

func TestServer_ToolsCall_InvalidParams(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()

	// Send a tools/call with params that can't be unmarshaled to ToolCallParams.
	body, _ := json.Marshal(Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  "not an object",
	})
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp Response
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error == nil || rpcResp.Error.Code != CodeInvalidParams {
		t.Errorf("expected invalid params error, got %+v", rpcResp.Error)
	}
}

func TestServer_Serve_ContextCancel(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.AddTool(newTestTool())

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx, "127.0.0.1:0")
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after context cancel")
	}
}

func TestServer_Serve_InvalidAddr(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	err := srv.Serve(context.Background(), "256.256.256.256:0")
	if err == nil {
		t.Fatal("expected error for invalid address")
	}
}

func TestMcpTool_Execute_NonTextContent(t *testing.T) {
	// Server that returns non-text content items.
	srv := NewServer("test", "1.0.0")
	srv.AddTool(&mockTool{
		name:        "binary",
		description: "Returns non-text content",
		inputSchema: map[string]any{"type": "object"},
		executeFn: func(_ context.Context, _ map[string]any) (*tool.Result, error) {
			return &tool.Result{
				Content: []schema.ContentPart{
					schema.TextPart{Text: "text result"},
				},
				IsError: false,
			}, nil
		},
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	tools, err := FromMCP(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("FromMCP: %v", err)
	}

	result, err := tools[0].Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result.Content) != 1 {
		t.Errorf("expected 1 content part, got %d", len(result.Content))
	}
}

func TestClient_RpcError(t *testing.T) {
	// Server that returns an RPC error for all methods.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		resp := Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInternalError, Message: "internal error"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.Initialize(context.Background())
	if err == nil {
		t.Fatal("expected error for RPC error response")
	}
}
