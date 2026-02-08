package mcp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

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
