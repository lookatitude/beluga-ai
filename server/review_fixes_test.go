package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
)

// FI-07: the stdlib adapter must strip the mount prefix so the agent's
// /invoke and /stream routes match.
func TestStdlibAdapter_RegisterAgent_PrefixRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		path   string
	}{
		{name: "nested prefix invoke", prefix: "/api/agent", path: "/api/agent/invoke"},
		{name: "trailing slash prefix", prefix: "/api/agent/", path: "/api/agent/invoke"},
		{name: "empty prefix", prefix: "", path: "/invoke"},
		{name: "nested prefix stream", prefix: "/v1/bot", path: "/v1/bot/stream"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ad := NewStdlibAdapter(Config{})
			a := &mockAgent{id: "x", result: "hi", events: []agent.Event{{Type: agent.EventText, Text: "hi"}}}
			if err := ad.RegisterAgent(tt.prefix, a); err != nil {
				t.Fatalf("RegisterAgent: %v", err)
			}
			rr := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, tt.path, strings.NewReader(`{"input":"x"}`))
			ad.mux.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %q", rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), "hi") {
				t.Errorf("body = %q, want agent output", rr.Body.String())
			}
		})
	}
}

// FI-21 / FI-05: internal error text (which can contain provider URLs with
// keys) must not be returned to clients.
func TestHandleInvoke_DoesNotLeakInternalError(t *testing.T) {
	secret := `Post "https://api.example.com/v1?key=SECRET-KEY-123": dial tcp: connection refused`
	a := &mockAgent{id: "x", err: core.Errorf(core.ErrProviderDown, "llm: %w", errors.New(secret))}
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/invoke", strings.NewReader(`{"input":"x"}`))
	NewAgentHandler(a).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rr.Code)
	}
	body := rr.Body.String()
	if strings.Contains(body, "SECRET-KEY-123") || strings.Contains(body, "api.example.com") {
		t.Fatalf("internal error leaked: %s", body)
	}
	var resp InvokeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != msgAgentFailed {
		t.Errorf("Error = %q, want %q", resp.Error, msgAgentFailed)
	}
	if resp.Code != string(core.ErrProviderDown) {
		t.Errorf("Code = %q, want %q", resp.Code, core.ErrProviderDown)
	}
	if len(resp.RequestID) != 32 {
		t.Errorf("RequestID = %q, want 32 hex chars", resp.RequestID)
	}
}

func TestHandleInvoke_PlainErrorHasNoCode(t *testing.T) {
	a := &mockAgent{id: "x", err: errors.New("boom")}
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/invoke", strings.NewReader(`{"input":"x"}`))
	NewAgentHandler(a).ServeHTTP(rr, req)
	var resp InvokeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != "" || strings.Contains(rr.Body.String(), "boom") {
		t.Errorf("unexpected response %s", rr.Body.String())
	}
}

func TestHandleInvoke_InvalidBodyDoesNotEchoDecoderError(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/invoke", strings.NewReader(`{"input": <script>`))
	NewAgentHandler(&mockAgent{id: "x"}).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
	if strings.Contains(rr.Body.String(), "invalid character") {
		t.Errorf("decoder detail leaked: %s", rr.Body.String())
	}
}

func TestHandleInvoke_BodyTooLarge(t *testing.T) {
	for _, path := range []string{"/invoke", "/stream"} {
		t.Run(path, func(t *testing.T) {
			big := `{"input":"` + strings.Repeat("a", MaxRequestBodyBytes+10) + `"}`
			rr := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(big))
			NewAgentHandler(&mockAgent{id: "x"}).ServeHTTP(rr, req)
			if rr.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want 413", rr.Code)
			}
		})
	}
}

// FI-47: SSE single-line fields must not allow field/event injection.
func TestSSEWriter_SanitizesFields(t *testing.T) {
	rr := httptest.NewRecorder()
	sw, err := NewSSEWriter(rr)
	if err != nil {
		t.Fatal(err)
	}
	if err := sw.WriteEvent(SSEEvent{
		ID:    "1\nevent: admin",
		Event: "text\r\ndata: injected",
		Data:  "a\r\nb\rc",
	}); err != nil {
		t.Fatal(err)
	}
	got := rr.Body.String()
	want := "id: 1event: admin\nevent: textdata: injected\ndata: a\ndata: b\ndata: c\n\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestStdlibAdapter_DefaultTimeouts(t *testing.T) {
	read, idle := NewStdlibAdapter(Config{}).effectiveTimeouts()
	if read != DefaultReadTimeout || idle != DefaultIdleTimeout {
		t.Errorf("defaults = %v/%v", read, idle)
	}
	read, idle = NewStdlibAdapter(Config{ReadTimeout: time.Second, IdleTimeout: 2 * time.Second}).effectiveTimeouts()
	if read != time.Second || idle != 2*time.Second {
		t.Errorf("explicit = %v/%v", read, idle)
	}
}
