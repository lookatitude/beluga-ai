package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/tool"
)

// rawResult is the status and headers of a rawPost call (body already closed).
type rawResult struct {
	StatusCode int
	Header     http.Header
}

func rawPost(t *testing.T, url, body string, hdr map[string]string) (rawResult, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return rawResult{StatusCode: resp.StatusCode, Header: resp.Header}, raw
}

func TestServer_NotificationGets202NoBody(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()
	for _, body := range []string{
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"notifications/unknown","params":{}}`,
	} {
		resp, raw := rawPost(t, ts.URL, body, nil)
		if resp.StatusCode != http.StatusAccepted || len(raw) != 0 {
			t.Fatalf("%s: got %d %q, want 202 with empty body", body, resp.StatusCode, raw)
		}
	}
	// Notification with a bad jsonrpc version is a 400.
	resp, _ := rawPost(t, ts.URL, `{"jsonrpc":"1.0","method":"x"}`, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad notification: status %d", resp.StatusCode)
	}
}

func TestServer_Ping(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()
	_, raw := rawPost(t, ts.URL, `{"jsonrpc":"2.0","id":7,"method":"ping"}`, nil)
	var resp struct {
		ID     float64         `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *RPCError       `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil || string(resp.Result) != "{}" || resp.ID != 7 {
		t.Fatalf("unexpected ping response %s", raw)
	}
}

func TestServer_ReadAndGetNotImplemented(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()
	for _, m := range []string{"resources/read", "prompts/get"} {
		_, raw := rawPost(t, ts.URL, `{"jsonrpc":"2.0","id":1,"method":"`+m+`"}`, nil)
		var resp Response
		_ = json.Unmarshal(raw, &resp)
		if resp.Error == nil || resp.Error.Code != CodeMethodNotFound {
			t.Fatalf("%s: expected method not found, got %s", m, raw)
		}
	}
}

func TestServer_InitializeNegotiatesVersion(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()
	tests := []struct{ req, want string }{
		{"2025-06-18", "2025-06-18"},
		{"2025-03-26", "2025-03-26"},
		{"1999-01-01", "2025-03-26"},
	}
	for _, tt := range tests {
		_, raw := rawPost(t, ts.URL, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"`+tt.req+`"}}`, nil)
		var resp struct {
			Result InitializeResult `json:"result"`
		}
		_ = json.Unmarshal(raw, &resp)
		if resp.Result.ProtocolVersion != tt.want {
			t.Errorf("requested %s: got %s, want %s", tt.req, resp.Result.ProtocolVersion, tt.want)
		}
	}
}

func TestServer_OriginAllowlist(t *testing.T) {
	srv := NewServer("s", "1", WithAllowedOrigins("https://app.example.com/"))
	srv.AddTool(newTestTool())
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	ping := `{"jsonrpc":"2.0","id":1,"method":"ping"}`

	tests := []struct {
		name, origin string
		want         int
	}{
		{"no origin", "", http.StatusOK},
		{"allowed", "https://app.example.com", http.StatusOK},
		{"allowed case-insensitive", "HTTPS://APP.example.com", http.StatusOK},
		{"rebinding", "http://evil.example", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := map[string]string{}
			if tt.origin != "" {
				hdr["Origin"] = tt.origin
			}
			resp, _ := rawPost(t, ts.URL, ping, hdr)
			if resp.StatusCode != tt.want {
				t.Fatalf("status %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}

	// Default server performs no Origin check (backwards compatible).
	_, dts := setupTestServer()
	defer dts.Close()
	if resp, _ := rawPost(t, dts.URL, ping, map[string]string{"Origin": "http://evil.example"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("default server: status %d", resp.StatusCode)
	}
}

func TestServer_BodyLimit(t *testing.T) {
	srv := NewServer("s", "1", WithMaxBodyBytes(64))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	resp, _ := rawPost(t, ts.URL, `{"jsonrpc":"2.0","id":1,"method":"ping","params":{"pad":"`+strings.Repeat("a", 200)+`"}}`, nil)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d, want 413", resp.StatusCode)
	}
	// Default limit is 1 MiB; non-positive option keeps it.
	if d := NewServer("s", "1", WithMaxBodyBytes(0)); d.maxBodyBytes != 1<<20 {
		t.Fatalf("default limit %d", d.maxBodyBytes)
	}
}

func TestServer_ErrorsDoNotLeakInternals(t *testing.T) {
	srv := NewServer("s", "1")
	srv.AddTool(&mockTool{name: "bad", executeFn: func(context.Context, map[string]any) (*tool.Result, error) {
		return nil, fmt.Errorf("dial postgres://user:SECRET@db")
	}})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_, raw := rawPost(t, ts.URL, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"bad"}}`, nil)
	if strings.Contains(string(raw), "SECRET") || strings.Contains(string(raw), "postgres") {
		t.Fatalf("tool error leaked: %s", raw)
	}
	var resp struct {
		Error struct {
			Code    int               `json:"code"`
			Message string            `json:"message"`
			Data    map[string]string `json:"data"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &resp)
	if resp.Error.Code != CodeInternalError || resp.Error.Message != "internal error" || resp.Error.Data["request_id"] == "" {
		t.Fatalf("unexpected error %s", raw)
	}

	resp2, raw2 := rawPost(t, ts.URL, `{"jsonrpc": nope`, nil)
	if resp2.StatusCode != http.StatusBadRequest || strings.Contains(string(raw2), "invalid character") {
		t.Fatalf("parse error leaked decoder detail: %d %s", resp2.StatusCode, raw2)
	}
}

func TestServer_NonPostIs405(t *testing.T) {
	_, ts := setupTestServer()
	defer ts.Close()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed || resp.Header.Get("Allow") != http.MethodPost {
		t.Fatalf("status %d allow %q", resp.StatusCode, resp.Header.Get("Allow"))
	}
}

// TestServer_ToolsCallReleasesLock: a running tool must not block AddTool.
func TestServer_ToolsCallReleasesLock(t *testing.T) {
	srv := NewServer("s", "1")
	started := make(chan struct{})
	release := make(chan struct{})
	srv.AddTool(&mockTool{name: "slow", executeFn: func(context.Context, map[string]any) (*tool.Result, error) {
		close(started)
		<-release
		return tool.TextResult("done"), nil
	}})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL,
			strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"slow"}}`))
		if resp, err := http.DefaultClient.Do(req); err == nil {
			_ = resp.Body.Close()
		}
	}()
	<-started

	added := make(chan struct{})
	go func() {
		srv.AddTool(newTestTool())
		close(added)
	}()
	select {
	case <-added:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("AddTool blocked while a tool was executing")
	}
	close(release)
	<-done
}
