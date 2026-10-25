package rest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/protocol/internal/httpsafe"
)

// RESTServer exposes Beluga agents as REST/SSE HTTP endpoints.
type RESTServer struct { //nolint:revive // exported name is public API; renaming to remove the stutter would break callers
	agents map[string]agent.Agent
	mu     sync.RWMutex
}

// NewServer creates a new REST server.
func NewServer() *RESTServer {
	return &RESTServer{
		agents: make(map[string]agent.Agent),
	}
}

// RegisterAgent registers an agent at the given path prefix.
func (s *RESTServer) RegisterAgent(path string, a agent.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path = strings.Trim(path, "/")
	if path == "" {
		return core.Errorf(core.ErrInvalidInput, "rest/register: path cannot be empty")
	}
	if _, exists := s.agents[path]; exists {
		return core.Errorf(core.ErrInvalidInput, "rest/register: path %q already registered", path)
	}
	s.agents[path] = a
	return nil
}

// Handler returns an http.Handler for the REST server.
func (s *RESTServer) Handler() http.Handler {
	return http.HandlerFunc(s.handleRequest)
}

// Serve starts the REST server on the given address. It blocks until the context
// is canceled or an error occurs.
//
// The server uses a 10s ReadHeaderTimeout, a 30s ReadTimeout and a 120s
// IdleTimeout. WriteTimeout is left at zero because SSE streams are
// long-lived. Request bodies are limited to 1 MiB.
func (s *RESTServer) Serve(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: httpsafe.DefaultReadHeaderTimeout,
		ReadTimeout:       httpsafe.DefaultReadTimeout,
		IdleTimeout:       httpsafe.DefaultIdleTimeout,
		// WriteTimeout is intentionally 0: the /stream endpoint serves
		// long-lived SSE responses that a write deadline would truncate.
		// Per-request lifetime is bounded by the request context instead.
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return core.Errorf(core.ErrProviderDown, "rest/serve: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		if shutdownErr := srv.Close(); shutdownErr != nil {
			return core.Errorf(core.ErrProviderDown, "rest/serve: shutdown: %w", shutdownErr)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return core.Errorf(core.ErrProviderDown, "rest/serve: %w", err)
	}
}

// InvokeRequest is the payload for synchronous invocation.
type InvokeRequest struct {
	Input string `json:"input"`
}

// InvokeResponse wraps the result of a synchronous invocation.
type InvokeResponse struct {
	Result string `json:"result"`
}

// StreamRequest is the payload for streaming invocation.
type StreamRequest struct {
	Input string `json:"input"`
}

// StreamEvent is an SSE-formatted agent event.
type StreamEvent struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	AgentID string `json:"agent_id,omitempty"`
	// RequestID is set on "error" events: it correlates the generic
	// client-facing message with the detailed server-side log entry.
	RequestID string `json:"request_id,omitempty"`
}

// errorResponse is the JSON error body returned by the REST server.
type errorResponse struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// writeJSONError writes a JSON error body with Content-Type
// application/json. The body is produced by json.Marshal, so message is
// always correctly escaped.
func writeJSONError(w http.ResponseWriter, status int, message, requestID string) {
	body, err := json.Marshal(errorResponse{Error: message, RequestID: requestID})
	if err != nil {
		body = []byte(`{"error":"internal error"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if requestID != "" {
		w.Header().Set(httpsafe.RequestIDHeader, requestID)
	}
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

// writeInternalError logs err server-side with a fresh correlation ID and
// returns a generic message carrying only that ID to the client.
func writeInternalError(w http.ResponseWriter, agentPath string, err error) {
	reqID := httpsafe.NewRequestID()
	slog.Error("rest: agent invoke failed",
		"request_id", reqID,
		"agent", httpsafe.LogValue(agentPath),
		"err", httpsafe.LogValue(err.Error()))
	writeJSONError(w, http.StatusInternalServerError, httpsafe.GenericInternalError, reqID)
}

// decodeBody decodes a JSON request body bounded by
// httpsafe.DefaultMaxBodyBytes. On failure it writes the error response
// (413 for oversized bodies, 400 otherwise) and returns false.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, httpsafe.DefaultMaxBodyBytes)).Decode(dst); err != nil {
		if httpsafe.IsBodyTooLarge(err) {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "request body too large", "")
			return false
		}
		writeJSONError(w, http.StatusBadRequest, "invalid request body", "")
		return false
	}
	return true
}

func (s *RESTServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	// Parse path: /{path}/{action} where action is "invoke" or "stream"
	path := strings.TrimPrefix(r.URL.Path, "/")
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash < 0 {
		writeJSONError(w, http.StatusNotFound, "invalid path, expected /{path}/{invoke|stream}", "")
		return
	}

	agentPath := path[:lastSlash]
	action := path[lastSlash+1:]

	s.mu.RLock()
	a, ok := s.agents[agentPath]
	s.mu.RUnlock()

	if !ok {
		writeJSONError(w, http.StatusNotFound, "agent not found", "")
		return
	}

	switch action {
	case "invoke":
		s.handleInvoke(r.Context(), w, r, agentPath, a)
	case "stream":
		s.handleStream(r.Context(), w, r, agentPath, a)
	default:
		writeJSONError(w, http.StatusNotFound, "unknown action, expected invoke or stream", "")
	}
}

func (s *RESTServer) handleInvoke(ctx context.Context, w http.ResponseWriter, r *http.Request, agentPath string, a agent.Agent) {
	var req InvokeRequest
	if !decodeBody(w, r, &req) {
		return
	}

	if req.Input == "" {
		writeJSONError(w, http.StatusBadRequest, "input is required", "")
		return
	}

	result, err := a.Invoke(ctx, req.Input)
	if err != nil {
		writeInternalError(w, agentPath, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Encode errors mean the client disconnected; nothing to do.
	_ = json.NewEncoder(w).Encode(InvokeResponse{Result: result})
}

func (s *RESTServer) handleStream(ctx context.Context, w http.ResponseWriter, r *http.Request, agentPath string, a agent.Agent) {
	var req StreamRequest
	if !decodeBody(w, r, &req) {
		return
	}

	if req.Input == "" {
		writeJSONError(w, http.StatusBadRequest, "input is required", "")
		return
	}

	sse, err := NewSSEWriter(w)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "streaming not supported", "")
		return
	}

	for event, err := range a.Stream(ctx, req.Input) {
		if err != nil {
			reqID := httpsafe.NewRequestID()
			slog.Error("rest: agent stream failed",
				"request_id", reqID,
				"agent", httpsafe.LogValue(agentPath),
				"err", httpsafe.LogValue(err.Error()))
			data, _ := json.Marshal(StreamEvent{Type: "error", Text: httpsafe.GenericInternalError, RequestID: reqID})
			// Ignore the write error: we are already in an error path
			// and about to return anyway.
			_ = sse.WriteEvent(SSEEvent{Event: "error", Data: string(data)})
			return
		}

		data, _ := json.Marshal(StreamEvent{
			Type:    string(event.Type),
			Text:    event.Text,
			AgentID: event.AgentID,
		})
		if sse.WriteEvent(SSEEvent{Event: string(event.Type), Data: string(data)}) != nil {
			return
		}
	}

	// Send a done event. Ignore write errors — the stream is ending.
	data, _ := json.Marshal(StreamEvent{Type: "done"})
	_ = sse.WriteEvent(SSEEvent{Event: "done", Data: string(data)})
}
