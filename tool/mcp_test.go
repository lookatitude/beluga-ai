package tool

import (
	"context"
	"testing"
)

func TestNewMCPClient_Defaults(t *testing.T) {
	c := NewMCPClient("http://localhost:8080")
	if c.serverURL != "http://localhost:8080" {
		t.Errorf("serverURL = %q, want %q", c.serverURL, "http://localhost:8080")
	}
	if c.opts.sessionID != "" {
		t.Errorf("sessionID = %q, want empty", c.opts.sessionID)
	}
	if c.opts.lastEventID != "" {
		t.Errorf("lastEventID = %q, want empty", c.opts.lastEventID)
	}
	if c.opts.headers == nil {
		t.Error("headers should be initialized (non-nil)")
	}
}

func TestNewMCPClient_WithSessionID(t *testing.T) {
	c := NewMCPClient("http://localhost:8080", WithSessionID("sess-123"))
	if c.opts.sessionID != "sess-123" {
		t.Errorf("sessionID = %q, want %q", c.opts.sessionID, "sess-123")
	}
}

func TestNewMCPClient_WithLastEventID(t *testing.T) {
	c := NewMCPClient("http://localhost:8080", WithLastEventID("evt-456"))
	if c.opts.lastEventID != "evt-456" {
		t.Errorf("lastEventID = %q, want %q", c.opts.lastEventID, "evt-456")
	}
}

func TestNewMCPClient_WithMCPHeaders(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer token",
		"X-Custom":      "value",
	}
	c := NewMCPClient("http://localhost:8080", WithMCPHeaders(headers))
	if c.opts.headers == nil {
		t.Fatal("headers should not be nil")
	}
	if c.opts.headers["Authorization"] != "Bearer token" {
		t.Errorf("Authorization header = %q, want %q", c.opts.headers["Authorization"], "Bearer token")
	}
	if c.opts.headers["X-Custom"] != "value" {
		t.Errorf("X-Custom header = %q, want %q", c.opts.headers["X-Custom"], "value")
	}
}

func TestNewMCPClient_AllOptions(t *testing.T) {
	c := NewMCPClient("http://localhost:8080",
		WithSessionID("sess"),
		WithLastEventID("evt"),
		WithMCPHeaders(map[string]string{"X-Key": "val"}),
	)
	if c.opts.sessionID != "sess" {
		t.Errorf("sessionID = %q, want %q", c.opts.sessionID, "sess")
	}
	if c.opts.lastEventID != "evt" {
		t.Errorf("lastEventID = %q, want %q", c.opts.lastEventID, "evt")
	}
	if c.opts.headers["X-Key"] != "val" {
		t.Errorf("X-Key header = %q, want %q", c.opts.headers["X-Key"], "val")
	}
}

func TestMCPClient_Connect_NotImplemented(t *testing.T) {
	c := NewMCPClient("http://localhost:8080")
	err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error for unimplemented Connect")
	}
}

func TestMCPClient_ListTools_NotImplemented(t *testing.T) {
	c := NewMCPClient("http://localhost:8080")
	tools, err := c.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error for unimplemented ListTools")
	}
	if tools != nil {
		t.Errorf("tools = %v, want nil", tools)
	}
}

func TestMCPClient_ExecuteTool_NotImplemented(t *testing.T) {
	c := NewMCPClient("http://localhost:8080")
	result, err := c.ExecuteTool(context.Background(), "test", nil)
	if err == nil {
		t.Fatal("expected error for unimplemented ExecuteTool")
	}
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}

func TestMCPClient_Close_NotImplemented(t *testing.T) {
	c := NewMCPClient("http://localhost:8080")
	err := c.Close(context.Background())
	if err == nil {
		t.Fatal("expected error for unimplemented Close")
	}
}

func TestFromMCP_NotImplemented(t *testing.T) {
	tools, err := FromMCP(context.Background(), "http://localhost:8080")
	if err == nil {
		t.Fatal("expected error for unimplemented FromMCP")
	}
	if tools != nil {
		t.Errorf("tools = %v, want nil", tools)
	}
}

func TestFromMCP_WithOptions_NotImplemented(t *testing.T) {
	tools, err := FromMCP(context.Background(), "http://localhost:8080",
		WithSessionID("s1"),
		WithLastEventID("e1"),
	)
	if err == nil {
		t.Fatal("expected error for unimplemented FromMCP")
	}
	if tools != nil {
		t.Errorf("tools = %v, want nil", tools)
	}
}

func TestMCPOption_WithSessionID(t *testing.T) {
	opts := mcpOptions{headers: make(map[string]string)}
	WithSessionID("abc")(&opts)
	if opts.sessionID != "abc" {
		t.Errorf("sessionID = %q, want %q", opts.sessionID, "abc")
	}
}

func TestMCPOption_WithLastEventID(t *testing.T) {
	opts := mcpOptions{headers: make(map[string]string)}
	WithLastEventID("def")(&opts)
	if opts.lastEventID != "def" {
		t.Errorf("lastEventID = %q, want %q", opts.lastEventID, "def")
	}
}

func TestMCPOption_WithMCPHeaders(t *testing.T) {
	opts := mcpOptions{headers: make(map[string]string)}
	WithMCPHeaders(map[string]string{"key": "value"})(&opts)
	if opts.headers["key"] != "value" {
		t.Errorf("headers[key] = %q, want %q", opts.headers["key"], "value")
	}
}
