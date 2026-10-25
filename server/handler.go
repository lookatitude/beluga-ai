package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
)

// MaxRequestBodyBytes caps the size of invoke and stream request bodies.
const MaxRequestBodyBytes = 1 << 20

// Client-facing error messages. Internal error text (which may contain
// provider URLs, credentials embedded in URLs, or stack details) is never
// sent to clients; it is logged server-side under the request ID instead.
const (
	msgInvalidBody  = "invalid request body"
	msgBodyTooLarge = "request body too large"
	msgAgentFailed  = "agent invocation failed"
)

// InvokeRequest is the JSON body for invoke endpoints.
type InvokeRequest struct {
	Input string `json:"input"`
}

// InvokeResponse is the JSON response for invoke endpoints.
type InvokeResponse struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
	// Code is the core.ErrorCode of a failed invocation, when known.
	Code string `json:"code,omitempty"`
	// RequestID correlates a client-visible error with the server log entry
	// that holds the full error detail.
	RequestID string `json:"request_id,omitempty"`
}

// StreamEvent is the SSE event data format sent during streaming.
type StreamEvent struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	AgentID  string         `json:"agent_id,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	// Code is the core.ErrorCode of an "error" event, when known.
	Code string `json:"code,omitempty"`
	// RequestID correlates an "error" event with the server log entry.
	RequestID string `json:"request_id,omitempty"`
}

// NewAgentHandler creates an http.Handler that exposes an agent via HTTP.
// It supports two sub-paths:
//   - POST {prefix}/invoke — synchronous invocation, returns JSON
//   - POST {prefix}/stream — SSE stream of agent events
func NewAgentHandler(a agent.Agent) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /invoke", func(w http.ResponseWriter, r *http.Request) {
		handleInvoke(w, r, a)
	})
	mux.HandleFunc("POST /stream", func(w http.ResponseWriter, r *http.Request) {
		handleStream(w, r, a)
	})
	return mux
}

func handleInvoke(w http.ResponseWriter, r *http.Request, a agent.Agent) {
	req, ok := decodeRequest(w, r)
	if !ok {
		return
	}

	result, err := a.Invoke(r.Context(), req.Input)
	if err != nil {
		id := logAgentError(r, "invoke", a, err)
		writeJSON(w, http.StatusInternalServerError, InvokeResponse{
			Error:     msgAgentFailed,
			Code:      errorCode(err),
			RequestID: id,
		})
		return
	}

	writeJSON(w, http.StatusOK, InvokeResponse{Result: result})
}

// decodeRequest reads a size-limited InvokeRequest body. On failure it writes
// a generic 400/413 response and returns false.
func decodeRequest(w http.ResponseWriter, r *http.Request) (InvokeRequest, bool) {
	var req InvokeRequest
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, InvokeResponse{Error: msgBodyTooLarge})
			return req, false
		}
		writeJSON(w, http.StatusBadRequest, InvokeResponse{Error: msgInvalidBody})
		return req, false
	}
	return req, true
}

// errorCode returns the core.ErrorCode carried by err, or "" when none.
func errorCode(err error) string {
	var ce *core.Error
	if errors.As(err, &ce) {
		return string(ce.Code)
	}
	return ""
}

// logAgentError records the full error server-side and returns the request ID
// that the client receives in place of the error text.
func logAgentError(r *http.Request, op string, a agent.Agent, err error) string {
	_, _, id := SanitizeError(r.Context(), op, a.ID(), err)
	return id
}

// SanitizeError logs err in full (via log/slog, under a fresh request ID) and
// returns what is safe to send to a client: a generic message, the
// core.ErrorCode when err carries one, and the request ID that correlates
// the client response with the server log entry. Transport adapters use it
// so internal error text (provider URLs, keys embedded in URLs, SQL, file
// paths) never reaches clients.
func SanitizeError(ctx context.Context, op, agentID string, err error) (message, code, requestID string) {
	requestID = newRequestID()
	slog.ErrorContext(ctx, "server: agent "+op+" failed",
		slog.String("request_id", requestID),
		slog.String("agent_id", agentID),
		slog.String("error", err.Error()),
	)
	return msgAgentFailed, errorCode(err), requestID
}

// newRequestID returns a random 128-bit hex identifier.
func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read never returns an error.
	return hex.EncodeToString(b[:])
}

func handleStream(w http.ResponseWriter, r *http.Request, a agent.Agent) {
	req, ok := decodeRequest(w, r)
	if !ok {
		return
	}

	sw, err := NewSSEWriter(w)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, InvokeResponse{
			Error: "streaming not supported",
		})
		return
	}

	for event, err := range a.Stream(r.Context(), req.Input) {
		if err != nil {
			id := logAgentError(r, "stream", a, err)
			errData, _ := json.Marshal(StreamEvent{
				Type:      "error",
				Text:      msgAgentFailed,
				Code:      errorCode(err),
				RequestID: id,
			})
			_ = sw.WriteEvent(SSEEvent{Event: "error", Data: string(errData)})
			return
		}

		se := StreamEvent{
			Type:     string(event.Type),
			Text:     event.Text,
			AgentID:  event.AgentID,
			Metadata: event.Metadata,
		}
		data, _ := json.Marshal(se)
		eventType := string(event.Type)
		if eventType == "" {
			eventType = "message"
		}
		if sw.WriteEvent(SSEEvent{Event: eventType, Data: string(data)}) != nil {
			return
		}
	}

	// Send a done event to signal end of stream.
	doneData, _ := json.Marshal(StreamEvent{Type: "done"})
	_ = sw.WriteEvent(SSEEvent{Event: "done", Data: string(doneData)})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Encode errors mean the client disconnected; nothing to do.
	_ = json.NewEncoder(w).Encode(v)
}
