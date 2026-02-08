package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewSSEWriter(t *testing.T) {
	t.Run("success with flusher", func(t *testing.T) {
		w := httptest.NewRecorder()
		sw, err := NewSSEWriter(w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sw == nil {
			t.Fatal("expected non-nil SSEWriter")
		}

		// Check headers
		if got := w.Header().Get("Content-Type"); got != "text/event-stream" {
			t.Errorf("Content-Type = %q, want %q", got, "text/event-stream")
		}
		if got := w.Header().Get("Cache-Control"); got != "no-cache" {
			t.Errorf("Cache-Control = %q, want %q", got, "no-cache")
		}
		if got := w.Header().Get("Connection"); got != "keep-alive" {
			t.Errorf("Connection = %q, want %q", got, "keep-alive")
		}
	})

	t.Run("error without flusher", func(t *testing.T) {
		w := &noFlushWriter{}
		_, err := NewSSEWriter(w)
		if err == nil {
			t.Fatal("expected error for non-flushing writer")
		}
	})
}

func TestSSEWriter_WriteEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    SSEEvent
		expected string
	}{
		{
			name:  "data only",
			event: SSEEvent{Data: "hello"},
			expected: "data: hello\n\n",
		},
		{
			name:  "event and data",
			event: SSEEvent{Event: "message", Data: "hello"},
			expected: "event: message\ndata: hello\n\n",
		},
		{
			name:  "id, event, and data",
			event: SSEEvent{ID: "1", Event: "message", Data: "hello"},
			expected: "id: 1\nevent: message\ndata: hello\n\n",
		},
		{
			name:  "with retry",
			event: SSEEvent{Event: "message", Data: "hello", Retry: 3000},
			expected: "event: message\nretry: 3000\ndata: hello\n\n",
		},
		{
			name:  "multi-line data",
			event: SSEEvent{Data: "line1\nline2\nline3"},
			expected: "data: line1\ndata: line2\ndata: line3\n\n",
		},
		{
			name:  "all fields",
			event: SSEEvent{ID: "42", Event: "update", Data: "payload", Retry: 5000},
			expected: "id: 42\nevent: update\nretry: 5000\ndata: payload\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			sw, err := NewSSEWriter(w)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := sw.WriteEvent(tt.event); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// The body includes the initial flush (empty) plus the event.
			body := w.Body.String()
			if !strings.Contains(body, tt.expected) {
				t.Errorf("body does not contain expected event.\ngot:\n%s\nwant substring:\n%s", body, tt.expected)
			}
		})
	}
}

func TestSSEWriter_WriteHeartbeat(t *testing.T) {
	w := httptest.NewRecorder()
	sw, err := NewSSEWriter(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := sw.WriteHeartbeat(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	expected := ": heartbeat\n\n"
	if !strings.Contains(body, expected) {
		t.Errorf("body does not contain heartbeat.\ngot:\n%s\nwant substring:\n%s", body, expected)
	}
}

// noFlushWriter implements http.ResponseWriter but NOT http.Flusher.
type noFlushWriter struct{}

func (w *noFlushWriter) Header() http.Header         { return http.Header{} }
func (w *noFlushWriter) Write(b []byte) (int, error)  { return len(b), nil }
func (w *noFlushWriter) WriteHeader(statusCode int)    {}
