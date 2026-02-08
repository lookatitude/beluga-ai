package server

import (
	"fmt"
	"net/http"
	"strings"
)

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	// Event is the event type field. If empty, the event type is omitted.
	Event string

	// Data is the event data payload. Multi-line data is split and each line
	// is sent with its own "data:" prefix per the SSE specification.
	Data string

	// ID is the event ID field. If empty, the id field is omitted.
	ID string

	// Retry is the reconnection time in milliseconds. If 0, the retry field
	// is omitted.
	Retry int
}

// SSEWriter writes Server-Sent Events to an http.ResponseWriter.
// It requires the underlying writer to implement http.Flusher.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter creates a new SSEWriter from an http.ResponseWriter. It returns
// an error if the writer does not support http.Flusher (required for SSE).
// It also sets the appropriate response headers for SSE.
func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("server/sse: response writer does not support flushing")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	return &SSEWriter{w: w, flusher: flusher}, nil
}

// WriteEvent writes a single SSE event to the stream and flushes it.
func (sw *SSEWriter) WriteEvent(event SSEEvent) error {
	var b strings.Builder

	if event.ID != "" {
		fmt.Fprintf(&b, "id: %s\n", event.ID)
	}
	if event.Event != "" {
		fmt.Fprintf(&b, "event: %s\n", event.Event)
	}
	if event.Retry > 0 {
		fmt.Fprintf(&b, "retry: %d\n", event.Retry)
	}

	// Data field: split on newlines per SSE spec.
	lines := strings.Split(event.Data, "\n")
	for _, line := range lines {
		fmt.Fprintf(&b, "data: %s\n", line)
	}

	// Blank line terminates the event.
	b.WriteString("\n")

	if _, err := fmt.Fprint(sw.w, b.String()); err != nil {
		return fmt.Errorf("server/sse: write error: %w", err)
	}
	sw.flusher.Flush()
	return nil
}

// WriteHeartbeat writes an SSE comment (":heartbeat\n\n") to keep the
// connection alive. This is useful for proxies that close idle connections.
func (sw *SSEWriter) WriteHeartbeat() error {
	if _, err := fmt.Fprint(sw.w, ": heartbeat\n\n"); err != nil {
		return fmt.Errorf("server/sse: heartbeat write error: %w", err)
	}
	sw.flusher.Flush()
	return nil
}
