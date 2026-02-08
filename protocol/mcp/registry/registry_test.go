package registry

import (
	"context"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/protocol/mcp"
	"github.com/lookatitude/beluga-ai/tool"
)

// mockMCPClient implements MCPClientInterface for testing.
type mockMCPClient struct {
	initErr  error
	listErr  error
	callErr  error
	caps     *mcp.ServerCapabilities
	tools    []mcp.ToolInfo
	callResp *mcp.ToolCallResult
}

func (m *mockMCPClient) Initialize(_ context.Context) (*mcp.ServerCapabilities, error) {
	if m.initErr != nil {
		return nil, m.initErr
	}
	return m.caps, nil
}

func (m *mockMCPClient) ListTools(_ context.Context) ([]mcp.ToolInfo, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.tools, nil
}

func (m *mockMCPClient) CallTool(_ context.Context, name string, args map[string]any) (*mcp.ToolCallResult, error) {
	if m.callErr != nil {
		return nil, m.callErr
	}
	return m.callResp, nil
}

func newTestRegistry(client *mockMCPClient) *Registry {
	reg := New()
	reg.clientFactory = func(url string) MCPClientInterface {
		return client
	}
	return reg
}

func TestAddAndListServers(t *testing.T) {
	reg := New()
	reg.AddServer("search", "http://localhost:8081/mcp", "ai", "search")
	reg.AddServer("code", "http://localhost:8082/mcp", "code")

	servers := reg.Servers()
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	if servers[0].Name != "search" {
		t.Errorf("expected 'search', got %q", servers[0].Name)
	}
	if servers[1].Name != "code" {
		t.Errorf("expected 'code', got %q", servers[1].Name)
	}
}

func TestRemoveServer(t *testing.T) {
	reg := New()
	reg.AddServer("search", "http://localhost:8081/mcp")
	reg.AddServer("code", "http://localhost:8082/mcp")
	reg.RemoveServer("search")

	servers := reg.Servers()
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].Name != "code" {
		t.Errorf("expected 'code', got %q", servers[0].Name)
	}
}

func TestRemoveServer_Nonexistent(t *testing.T) {
	reg := New()
	reg.AddServer("search", "http://localhost:8081/mcp")
	reg.RemoveServer("nonexistent")

	servers := reg.Servers()
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
}

func TestServersByTag(t *testing.T) {
	reg := New()
	reg.AddServer("search", "http://localhost:8081/mcp", "ai", "search")
	reg.AddServer("code", "http://localhost:8082/mcp", "code")
	reg.AddServer("docs", "http://localhost:8083/mcp", "ai", "docs")

	aiServers := reg.ServersByTag("ai")
	if len(aiServers) != 2 {
		t.Fatalf("expected 2 ai servers, got %d", len(aiServers))
	}

	codeServers := reg.ServersByTag("code")
	if len(codeServers) != 1 {
		t.Fatalf("expected 1 code server, got %d", len(codeServers))
	}

	noneServers := reg.ServersByTag("missing")
	if len(noneServers) != 0 {
		t.Fatalf("expected 0 servers, got %d", len(noneServers))
	}
}

func TestDiscoverTools(t *testing.T) {
	client := &mockMCPClient{
		caps: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapability{},
		},
		tools: []mcp.ToolInfo{
			{Name: "search", Description: "Web search"},
			{Name: "calculate", Description: "Math operations"},
		},
	}
	reg := newTestRegistry(client)
	reg.AddServer("toolserver", "http://localhost:8081/mcp")

	discovered, err := reg.DiscoverTools(context.Background())
	if err != nil {
		t.Fatalf("DiscoverTools: %v", err)
	}

	if len(discovered) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(discovered))
	}
	if discovered[0].Tool.Name() != "search" {
		t.Errorf("expected 'search', got %q", discovered[0].Tool.Name())
	}
	if discovered[0].ServerName != "toolserver" {
		t.Errorf("expected 'toolserver', got %q", discovered[0].ServerName)
	}
	if discovered[1].Tool.Name() != "calculate" {
		t.Errorf("expected 'calculate', got %q", discovered[1].Tool.Name())
	}
}

func TestDiscoverTools_MultipleServers(t *testing.T) {
	client := &mockMCPClient{
		caps:  &mcp.ServerCapabilities{},
		tools: []mcp.ToolInfo{{Name: "tool1"}},
	}
	reg := newTestRegistry(client)
	reg.AddServer("server1", "http://localhost:8081/mcp")
	reg.AddServer("server2", "http://localhost:8082/mcp")

	discovered, err := reg.DiscoverTools(context.Background())
	if err != nil {
		t.Fatalf("DiscoverTools: %v", err)
	}

	if len(discovered) != 2 {
		t.Fatalf("expected 2 tools (1 from each), got %d", len(discovered))
	}
}

func TestDiscoverTools_InitError(t *testing.T) {
	client := &mockMCPClient{
		initErr: fmt.Errorf("connection refused"),
	}
	reg := newTestRegistry(client)
	reg.AddServer("server1", "http://localhost:8081/mcp")

	_, err := reg.DiscoverTools(context.Background())
	if err == nil {
		t.Fatal("expected error when all servers fail")
	}
}

func TestDiscoverTools_PartialFailure(t *testing.T) {
	// Use different clients for different URLs.
	clients := map[string]*mockMCPClient{
		"http://fail/mcp": {initErr: fmt.Errorf("connection refused")},
		"http://ok/mcp": {
			caps:  &mcp.ServerCapabilities{},
			tools: []mcp.ToolInfo{{Name: "ok-tool"}},
		},
	}

	reg := New()
	reg.clientFactory = func(url string) MCPClientInterface {
		if c, ok := clients[url]; ok {
			return c
		}
		return &mockMCPClient{initErr: fmt.Errorf("unknown")}
	}
	reg.AddServer("fail", "http://fail/mcp")
	reg.AddServer("ok", "http://ok/mcp")

	discovered, err := reg.DiscoverTools(context.Background())
	if err != nil {
		t.Fatalf("DiscoverTools: %v (should succeed with partial results)", err)
	}
	if len(discovered) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(discovered))
	}
}

func TestDiscoverToolsFromServer(t *testing.T) {
	client := &mockMCPClient{
		caps:  &mcp.ServerCapabilities{},
		tools: []mcp.ToolInfo{{Name: "tool1"}},
	}
	reg := newTestRegistry(client)
	reg.AddServer("myserver", "http://localhost:8081/mcp")

	discovered, err := reg.DiscoverToolsFromServer(context.Background(), "myserver")
	if err != nil {
		t.Fatalf("DiscoverToolsFromServer: %v", err)
	}
	if len(discovered) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(discovered))
	}
}

func TestDiscoverToolsFromServer_NotFound(t *testing.T) {
	reg := New()
	_, err := reg.DiscoverToolsFromServer(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestTools(t *testing.T) {
	client := &mockMCPClient{
		caps:  &mcp.ServerCapabilities{},
		tools: []mcp.ToolInfo{{Name: "search", Description: "Search tool"}},
	}
	reg := newTestRegistry(client)
	reg.AddServer("server1", "http://localhost:8081/mcp")

	tools, err := reg.Tools(context.Background())
	if err != nil {
		t.Fatalf("Tools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name() != "search" {
		t.Errorf("expected 'search', got %q", tools[0].Name())
	}
}

func TestRemoteTool_Execute(t *testing.T) {
	client := &mockMCPClient{
		callResp: &mcp.ToolCallResult{
			Content: []mcp.ContentItem{
				{Type: "text", Text: "result text"},
			},
		},
	}

	rt := newRemoteTool(client, mcp.ToolInfo{
		Name:        "search",
		Description: "Web search",
		InputSchema: map[string]any{"type": "object"},
	})

	result, err := rt.Execute(context.Background(), map[string]any{"q": "test"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.IsError {
		t.Error("expected IsError=false")
	}
}

func TestRemoteTool_ExecuteError(t *testing.T) {
	client := &mockMCPClient{
		callErr: fmt.Errorf("tool failed"),
	}

	rt := newRemoteTool(client, mcp.ToolInfo{Name: "search"})

	_, err := rt.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRemoteTool_Metadata(t *testing.T) {
	rt := newRemoteTool(nil, mcp.ToolInfo{
		Name:        "calculator",
		Description: "Perform math",
		InputSchema: map[string]any{"type": "object"},
	})

	if rt.Name() != "calculator" {
		t.Errorf("expected 'calculator', got %q", rt.Name())
	}
	if rt.Description() != "Perform math" {
		t.Errorf("expected 'Perform math', got %q", rt.Description())
	}
	if rt.InputSchema()["type"] != "object" {
		t.Errorf("expected type 'object', got %v", rt.InputSchema()["type"])
	}
}

func TestInterfaceCompliance(t *testing.T) {
	var _ tool.Tool = (*remoteTool)(nil)
}
