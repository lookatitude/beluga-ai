package rest

import (
	"fmt"
	"net/http"
)

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	// Event is the event type. If empty, no event line is written.
	Event string `json:"event,omitempty"`
	// Data is the event data payload.
	Data string `json:"data"`
	// ID is the event ID. If empty, no id line is written.
	ID string `json:"id,omitempty"`
}

// SSEWriter writes Server-Sent Events to an HTTP response.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter creates a new SSEWriter. It returns an error if the
// ResponseWriter does not support flushing.
func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("rest/sse: response writer does not support flushing")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	return &SSEWriter{w: w, flusher: flusher}, nil
}

// WriteEvent writes a single SSE event and flushes the response.
func (sw *SSEWriter) WriteEvent(event SSEEvent) error {
	if event.ID != "" {
		if _, err := fmt.Fprintf(sw.w, "id: %s\n", event.ID); err != nil {
			return fmt.Errorf("rest/sse: write id: %w", err)
		}
	}
	if event.Event != "" {
		if _, err := fmt.Fprintf(sw.w, "event: %s\n", event.Event); err != nil {
			return fmt.Errorf("rest/sse: write event: %w", err)
		}
	}
	if _, err := fmt.Fprintf(sw.w, "data: %s\n\n", event.Data); err != nil {
		return fmt.Errorf("rest/sse: write data: %w", err)
	}
	sw.flusher.Flush()
	return nil
}

// WriteHeartbeat writes a comment line as a keep-alive heartbeat.
func (sw *SSEWriter) WriteHeartbeat() error {
	if _, err := fmt.Fprint(sw.w, ": heartbeat\n\n"); err != nil {
		return fmt.Errorf("rest/sse: write heartbeat: %w", err)
	}
	sw.flusher.Flush()
	return nil
}
