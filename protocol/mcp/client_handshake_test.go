package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordedReq captures what the fake streamable-HTTP server received.
type recordedReq struct {
	method    string
	params    map[string]any
	hasID     bool
	accept    string
	sessionID string
	version   string
}

// fakeStreamableServer is a minimal 2025-06-18 server that answers via SSE,
// assigns a session id, and requires it on subsequent requests.
type fakeStreamableServer struct {
	mu   sync.Mutex
	reqs []recordedReq
	sse  bool
}

func (f *fakeStreamableServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var msg struct {
		ID     any            `json:"id"`
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
	}
	_ = json.NewDecoder(r.Body).Decode(&msg)
	f.mu.Lock()
	f.reqs = append(f.reqs, recordedReq{
		method: msg.Method, params: msg.Params, hasID: msg.ID != nil,
		accept: r.Header.Get("Accept"), sessionID: r.Header.Get("Mcp-Session-Id"),
		version: r.Header.Get("MCP-Protocol-Version"),
	})
	f.mu.Unlock()

	if msg.Method != "initialize" && r.Header.Get("Mcp-Session-Id") != "sess-42" {
		http.Error(w, "missing session", http.StatusBadRequest)
		return
	}
	if msg.ID == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	var result any
	switch msg.Method {
	case "initialize":
		w.Header().Set("Mcp-Session-Id", "sess-42")
		result = map[string]any{
			"protocolVersion": "2025-06-18",
			"serverInfo":      map[string]any{"name": "fake", "version": "1"},
			"capabilities":    map[string]any{"tools": map[string]any{}},
		}
	case "tools/list":
		result = map[string]any{"tools": []map[string]any{{"name": "t1", "inputSchema": map[string]any{"type": "object"}}}}
	case "tools/call":
		result = map[string]any{"content": []map[string]any{{"type": "text", "text": "hi"}}}
	}
	resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": result})
	if !f.sse {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(resp)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	// A server notification and an unrelated response precede the answer.
	fmt.Fprintf(w, ": comment\n\n")
	fmt.Fprintf(w, "event: message\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\",\"params\":{}}\n\n")
	fmt.Fprintf(w, "data: {\"jsonrpc\":\"2.0\",\"id\":9999,\"result\":{}}\n\n")
	fmt.Fprintf(w, "event: message\r\ndata: %s\r\n\r\n", resp)
}

func TestClient_StreamableHTTPHandshake(t *testing.T) {
	for _, sse := range []bool{false, true} {
		t.Run(fmt.Sprintf("sse=%v", sse), func(t *testing.T) {
			fake := &fakeStreamableServer{sse: sse}
			ts := httptest.NewServer(fake)
			defer ts.Close()

			tools, err := FromMCP(context.Background(), ts.URL)
			if err != nil {
				t.Fatalf("FromMCP: %v", err)
			}
			if len(tools) != 1 || tools[0].Name() != "t1" {
				t.Fatalf("tools = %v", tools)
			}
			res, err := tools[0].Execute(context.Background(), map[string]any{})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if len(res.Content) != 1 {
				t.Fatalf("content = %v", res.Content)
			}

			fake.mu.Lock()
			defer fake.mu.Unlock()
			wantOrder := []string{"initialize", "notifications/initialized", "tools/list", "tools/call"}
			if len(fake.reqs) != len(wantOrder) {
				t.Fatalf("requests = %+v", fake.reqs)
			}
			for i, r := range fake.reqs {
				if r.method != wantOrder[i] {
					t.Errorf("req %d method %q, want %q", i, r.method, wantOrder[i])
				}
				if r.accept != "application/json, text/event-stream" {
					t.Errorf("req %d Accept %q", i, r.accept)
				}
				if i > 0 && (r.sessionID != "sess-42" || r.version != "2025-06-18") {
					t.Errorf("req %d session %q version %q", i, r.sessionID, r.version)
				}
			}
			if fake.reqs[1].hasID {
				t.Error("notifications/initialized must not carry an id")
			}
			p := fake.reqs[0].params
			if p["protocolVersion"] != "2025-06-18" {
				t.Errorf("protocolVersion = %v", p["protocolVersion"])
			}
			if _, ok := p["capabilities"].(map[string]any); !ok {
				t.Errorf("capabilities missing: %v", p)
			}
			ci, _ := p["clientInfo"].(map[string]any)
			if ci["name"] == "" || ci["name"] == nil || ci["version"] == nil {
				t.Errorf("clientInfo = %v", ci)
			}
		})
	}
}

func TestClient_SessionIDAndClientInfo(t *testing.T) {
	fake := &fakeStreamableServer{}
	ts := httptest.NewServer(fake)
	defer ts.Close()
	c := NewClient(ts.URL, WithClientInfo("my-app", "9.9"), WithClientHTTPClient(nil))
	if c.SessionID() != "" {
		t.Fatal("session id before init")
	}
	if _, err := c.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c.SessionID() != "sess-42" {
		t.Fatalf("SessionID = %q", c.SessionID())
	}
	ci := fake.reqs[0].params["clientInfo"].(map[string]any)
	if ci["name"] != "my-app" || ci["version"] != "9.9" {
		t.Fatalf("clientInfo = %v", ci)
	}
}

func TestClient_DefaultHTTPClientHasTimeout(t *testing.T) {
	c := NewClient("http://example.com")
	if c.httpClient == http.DefaultClient || c.httpClient.Timeout != DefaultClientTimeout {
		t.Fatalf("expected dedicated client with %v timeout, got %v", DefaultClientTimeout, c.httpClient.Timeout)
	}
	hc := &http.Client{Timeout: time.Second}
	if NewClient("http://x", WithClientHTTPClient(hc)).httpClient != hc {
		t.Fatal("WithClientHTTPClient ignored")
	}
}

func TestClient_ResponseBodyIsBounded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"`))
		chunk := []byte(strings.Repeat("a", 1<<20))
		for i := 0; i < 12; i++ {
			_, _ = w.Write(chunk)
		}
		_, _ = w.Write([]byte(`"}]}}`))
	}))
	defer ts.Close()
	if _, err := NewClient(ts.URL).ListTools(context.Background()); err == nil {
		t.Fatal("expected error for response over the size limit")
	}
}

func TestClient_SSEWithoutMatchingResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		fmt.Fprint(w, "data: {\"jsonrpc\":\"2.0\",\"id\":555,\"result\":{}}\n\ndata: not json\n\n")
	}))
	defer ts.Close()
	_, err := NewClient(ts.URL).ListTools(context.Background())
	if err == nil || !strings.Contains(err.Error(), "without a response") {
		t.Fatalf("err = %v", err)
	}
}

func TestClient_SSEFinalEventWithoutBlankLine(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"jsonrpc\":\"2.0\",\"id\":\"1\",\n")
		fmt.Fprint(w, "data: \"result\":{\"tools\":[]}}")
	}))
	defer ts.Close()
	if _, err := NewClient(ts.URL).ListTools(context.Background()); err != nil {
		t.Fatalf("ListTools: %v", err)
	}
}

func TestClient_HTTPErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer ts.Close()
	_, err := NewClient(ts.URL).ListTools(context.Background())
	if err == nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Fatalf("err = %v", err)
	}
	// JSON body on a non-2xx status without an RPC error still fails.
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{}}`)
	}))
	defer ts2.Close()
	if _, err := NewClient(ts2.URL).ListTools(context.Background()); err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("err = %v", err)
	}
}

func TestClient_InitializedNotificationFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg Request
		_ = json.NewDecoder(r.Body).Decode(&msg)
		if msg.ID == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{}}}`)
	}))
	defer ts.Close()
	if _, err := NewClient(ts.URL).Initialize(context.Background()); err == nil {
		t.Fatal("expected error when notifications/initialized is rejected")
	}
}

// TestClient_AgainstNativeServer: the native client and server interoperate.
func TestClient_AgainstNativeServer(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()
	c := NewClient(ts.URL)
	if _, err := c.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c.protocolVersion != ProtocolVersionLatest {
		t.Fatalf("negotiated %q", c.protocolVersion)
	}
}

func TestIDEquals(t *testing.T) {
	if !idEquals(float64(3), 3) || !idEquals("3", 3) || idEquals(nil, 3) || idEquals(true, 3) {
		t.Fatal("idEquals mismatch")
	}
}
