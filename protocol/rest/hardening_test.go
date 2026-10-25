package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/agent"
)

const secretErr = `provider failed: Post "https://api.example/v1?key=SECRET-KEY-123": "quoted"`

func newHardeningServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := NewServer()
	if err := srv.RegisterAgent("ok", &mockAgent{id: "ok"}); err != nil {
		t.Fatal(err)
	}
	if err := srv.RegisterAgent("fail", &mockAgent{
		id: "fail",
		invokeFn: func(context.Context, string) (string, error) {
			return "", fmt.Errorf("%s", secretErr)
		},
		streamFn: func(context.Context, string) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{}, fmt.Errorf("%s", secretErr))
			}
		},
	}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// TestRESTServer_InternalErrorIsSanitized: agent error text must not reach the
// client; the body must be valid JSON with a correlation ID.
func TestRESTServer_InternalErrorIsSanitized(t *testing.T) {
	ts := newHardeningServer(t)
	resp, err := testPost(t, ts.URL+"/fail/invoke", "application/json", strings.NewReader(`{"input":"hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if strings.Contains(string(raw), "SECRET") || strings.Contains(string(raw), "provider failed") {
		t.Fatalf("internal error leaked: %s", raw)
	}
	var body errorResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("body is not JSON: %v: %s", err, raw)
	}
	if body.Error != "internal error" || body.RequestID == "" {
		t.Errorf("unexpected body %+v", body)
	}
	if resp.Header.Get("X-Request-Id") != body.RequestID {
		t.Errorf("X-Request-Id header %q != body %q", resp.Header.Get("X-Request-Id"), body.RequestID)
	}
}

func TestRESTServer_StreamErrorIsSanitized(t *testing.T) {
	ts := newHardeningServer(t)
	resp, err := testPost(t, ts.URL+"/fail/stream", "application/json", strings.NewReader(`{"input":"hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(raw), "SECRET") {
		t.Fatalf("internal error leaked in SSE: %s", raw)
	}
	if !strings.Contains(string(raw), `"text":"internal error"`) || !strings.Contains(string(raw), `"request_id":"`) {
		t.Fatalf("expected generic error with request_id, got %s", raw)
	}
}

// TestRESTServer_ErrorBodiesAreJSON: every error path uses application/json
// bodies produced by json.Marshal.
func TestRESTServer_ErrorBodiesAreJSON(t *testing.T) {
	ts := newHardeningServer(t)
	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"method", http.MethodGet, "/ok/invoke", "", http.StatusMethodNotAllowed},
		{"no action", http.MethodPost, "/ok", "{}", http.StatusNotFound},
		{"agent missing", http.MethodPost, "/nope/invoke", "{}", http.StatusNotFound},
		{"bad action", http.MethodPost, "/ok/zap", "{}", http.StatusNotFound},
		{"bad json invoke", http.MethodPost, "/ok/invoke", "{not json", http.StatusBadRequest},
		{"bad json stream", http.MethodPost, "/ok/stream", "{not json", http.StatusBadRequest},
		{"empty input invoke", http.MethodPost, "/ok/invoke", `{"input":""}`, http.StatusBadRequest},
		{"empty input stream", http.MethodPost, "/ok/stream", `{"input":""}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequestWithContext(t.Context(), tt.method, ts.URL+tt.path, strings.NewReader(tt.body))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.want)
			}
			if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q", ct)
			}
			var body errorResponse
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.Error == "" {
				t.Errorf("invalid JSON error body: %v %+v", err, body)
			}
			if strings.Contains(body.Error, "invalid character") {
				t.Errorf("decoder detail leaked: %q", body.Error)
			}
		})
	}
}

func TestRESTServer_BodyLimit(t *testing.T) {
	ts := newHardeningServer(t)
	big := `{"input":"` + strings.Repeat("a", 2<<20) + `"}`
	for _, action := range []string{"invoke", "stream"} {
		resp, err := testPost(t, ts.URL+"/ok/"+action, "application/json", strings.NewReader(big))
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Errorf("%s: status = %d, want 413", action, resp.StatusCode)
		}
	}
}

func TestRESTServer_NonFlusherReturnsJSON(t *testing.T) {
	srv := NewServer()
	_ = srv.RegisterAgent("ok", &mockAgent{id: "ok"})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/ok/stream", strings.NewReader(`{"input":"x"}`))
	w := &recordingNonFlusher{h: http.Header{}}
	srv.handleRequest(w, req)
	if w.code != http.StatusInternalServerError || !strings.Contains(w.body.String(), `"streaming not supported"`) {
		t.Fatalf("got %d %q", w.code, w.body.String())
	}
}

type recordingNonFlusher struct {
	h    http.Header
	code int
	body strings.Builder
}

func (w *recordingNonFlusher) Header() http.Header         { return w.h }
func (w *recordingNonFlusher) Write(b []byte) (int, error) { return w.body.Write(b) }
func (w *recordingNonFlusher) WriteHeader(c int)           { w.code = c }

// TestSSEWriter_SanitizesFields: CR/LF in event/id must not inject fields;
// multi-line data becomes multiple data: lines.
func TestSSEWriter_SanitizesFields(t *testing.T) {
	rec := httptest.NewRecorder()
	sw, err := NewSSEWriter(rec)
	if err != nil {
		t.Fatal(err)
	}
	err = sw.WriteEvent(SSEEvent{
		ID:    "1\r\nevent: injected",
		Event: "text\ndata: injected",
		Data:  "line1\nline2\r\nevent: x",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "id: 1event: injected\nevent: textdata: injected\ndata: line1\ndata: line2\ndata: event: x\n\n"
	if got := rec.Body.String(); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
