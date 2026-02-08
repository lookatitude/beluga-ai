package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lookatitude/beluga-ai/agent"
)

// InvokeRequest is the JSON body for invoke endpoints.
type InvokeRequest struct {
	Input string `json:"input"`
}

// InvokeResponse is the JSON response for invoke endpoints.
type InvokeResponse struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

// StreamEvent is the SSE event data format sent during streaming.
type StreamEvent struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	AgentID  string         `json:"agent_id,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
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
	var req InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, InvokeResponse{
			Error: fmt.Sprintf("invalid request body: %v", err),
		})
		return
	}

	result, err := a.Invoke(r.Context(), req.Input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, InvokeResponse{
			Error: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, InvokeResponse{Result: result})
}

func handleStream(w http.ResponseWriter, r *http.Request, a agent.Agent) {
	var req InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, InvokeResponse{
			Error: fmt.Sprintf("invalid request body: %v", err),
		})
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
			errData, _ := json.Marshal(StreamEvent{Type: "error", Text: err.Error()})
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
		if writeErr := sw.WriteEvent(SSEEvent{Event: eventType, Data: string(data)}); writeErr != nil {
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
	json.NewEncoder(w).Encode(v)
}
