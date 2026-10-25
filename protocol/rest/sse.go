package rest

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/protocol/internal/httpsafe"
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
		return nil, core.Errorf(core.ErrInvalidInput, "rest/sse: response writer does not support flushing")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	return &SSEWriter{w: w, flusher: flusher}, nil
}

// WriteEvent writes a single SSE event and flushes the response.
//
// CR and LF are stripped from the single-line ID and Event fields so that a
// value containing a newline cannot inject extra SSE fields or events.
// Multi-line Data is emitted as one "data:" line per line, per the SSE
// specification, so embedded newlines cannot start a new field either.
func (sw *SSEWriter) WriteEvent(event SSEEvent) error {
	event.ID = httpsafe.SSEField(event.ID)
	event.Event = httpsafe.SSEField(event.Event)
	if event.ID != "" {
		if _, err := fmt.Fprintf(sw.w, "id: %s\n", event.ID); err != nil {
			return core.Errorf(core.ErrProviderDown, "rest/sse: write id: %w", err)
		}
	}
	if event.Event != "" {
		if _, err := fmt.Fprintf(sw.w, "event: %s\n", event.Event); err != nil {
			return core.Errorf(core.ErrProviderDown, "rest/sse: write event: %w", err)
		}
	}
	data := strings.ReplaceAll(strings.ReplaceAll(event.Data, "\r\n", "\n"), "\r", "\n")
	var b strings.Builder
	for _, line := range strings.Split(data, "\n") {
		b.WriteString("data: ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	if _, err := io.WriteString(sw.w, b.String()); err != nil {
		return core.Errorf(core.ErrProviderDown, "rest/sse: write data: %w", err)
	}
	sw.flusher.Flush()
	return nil
}

// WriteHeartbeat writes a comment line as a keep-alive heartbeat.
func (sw *SSEWriter) WriteHeartbeat() error {
	if _, err := fmt.Fprint(sw.w, ": heartbeat\n\n"); err != nil {
		return core.Errorf(core.ErrProviderDown, "rest/sse: write heartbeat: %w", err)
	}
	sw.flusher.Flush()
	return nil
}
