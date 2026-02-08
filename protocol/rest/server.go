// Package rest provides a REST/SSE API server for exposing Beluga agents over HTTP.
// It supports both synchronous invocation and streaming via Server-Sent Events.
//
// Usage:
//
//	srv := rest.NewServer()
//	srv.RegisterAgent("assistant", myAgent)
//	srv.Serve(ctx, ":8080")
//
//	// POST /assistant/invoke  -> {"result": "..."}
//	// POST /assistant/stream  -> SSE stream of events
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/agent"
)

// RESTServer exposes Beluga agents as REST/SSE HTTP endpoints.
type RESTServer struct {
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
		return fmt.Errorf("rest/register: path cannot be empty")
	}
	if _, exists := s.agents[path]; exists {
		return fmt.Errorf("rest/register: path %q already registered", path)
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
func (s *RESTServer) Serve(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("rest/serve: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		if shutdownErr := srv.Close(); shutdownErr != nil {
			return fmt.Errorf("rest/serve: shutdown: %w", shutdownErr)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("rest/serve: %w", err)
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
}

func (s *RESTServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /{path}/{action} where action is "invoke" or "stream"
	path := strings.TrimPrefix(r.URL.Path, "/")
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash < 0 {
		http.Error(w, `{"error":"invalid path, expected /{path}/{invoke|stream}"}`, http.StatusNotFound)
		return
	}

	agentPath := path[:lastSlash]
	action := path[lastSlash+1:]

	s.mu.RLock()
	a, ok := s.agents[agentPath]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"agent not found"}`, http.StatusNotFound)
		return
	}

	switch action {
	case "invoke":
		s.handleInvoke(r.Context(), w, r, a)
	case "stream":
		s.handleStream(r.Context(), w, r, a)
	default:
		http.Error(w, `{"error":"unknown action, expected invoke or stream"}`, http.StatusNotFound)
	}
}

func (s *RESTServer) handleInvoke(ctx context.Context, w http.ResponseWriter, r *http.Request, a agent.Agent) {
	var req InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Input == "" {
		http.Error(w, `{"error":"input is required"}`, http.StatusBadRequest)
		return
	}

	result, err := a.Invoke(ctx, req.Input)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(InvokeResponse{Result: result})
}

func (s *RESTServer) handleStream(ctx context.Context, w http.ResponseWriter, r *http.Request, a agent.Agent) {
	var req StreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Input == "" {
		http.Error(w, `{"error":"input is required"}`, http.StatusBadRequest)
		return
	}

	sse, err := NewSSEWriter(w)
	if err != nil {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}

	for event, err := range a.Stream(ctx, req.Input) {
		if err != nil {
			data, _ := json.Marshal(StreamEvent{Type: "error", Text: err.Error()})
			sse.WriteEvent(SSEEvent{Event: "error", Data: string(data)})
			return
		}

		data, _ := json.Marshal(StreamEvent{
			Type:    string(event.Type),
			Text:    event.Text,
			AgentID: event.AgentID,
		})
		if writeErr := sse.WriteEvent(SSEEvent{Event: string(event.Type), Data: string(data)}); writeErr != nil {
			return
		}
	}

	// Send a done event.
	data, _ := json.Marshal(StreamEvent{Type: "done"})
	sse.WriteEvent(SSEEvent{Event: "done", Data: string(data)})
}
