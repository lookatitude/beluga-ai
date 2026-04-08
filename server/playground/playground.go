package playground

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/agent"
)

// AgentSelector provides a list of available agents and retrieves them by ID.
type AgentSelector interface {
	// List returns the IDs of all available agents.
	List(ctx context.Context) []string
	// Get returns the agent with the given ID, or nil if not found.
	Get(ctx context.Context, id string) agent.Agent
}

// StreamAdapter converts an agent's streaming output to a format suitable
// for Server-Sent Events delivery.
type StreamAdapter interface {
	// WriteEvents reads events from the agent stream and writes them
	// as SSE to the http.ResponseWriter.
	WriteEvents(ctx context.Context, w http.ResponseWriter, a agent.Agent, input string) error
}

// Option configures the PlaygroundHandler.
type Option func(*options)

type options struct {
	title string
	path  string
}

// WithTitle sets the page title.
func WithTitle(t string) Option {
	return func(o *options) { o.title = t }
}

// WithBasePath sets the URL base path (default "/playground").
func WithBasePath(p string) Option {
	return func(o *options) { o.path = p }
}

// PlaygroundHandler serves the chat UI and handles API requests.
type PlaygroundHandler struct {
	selector AgentSelector
	adapter  StreamAdapter
	opts     options
}

// NewHandler creates a PlaygroundHandler serving a chat UI for the given agents.
func NewHandler(selector AgentSelector, opts ...Option) *PlaygroundHandler {
	o := options{
		title: "Beluga AI Playground",
		path:  "/playground",
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &PlaygroundHandler{
		selector: selector,
		adapter:  &defaultStreamAdapter{},
		opts:     o,
	}
}

// ServeHTTP implements http.Handler.
func (h *PlaygroundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.handleUI)
	mux.HandleFunc("GET /agents", h.handleListAgents)
	mux.HandleFunc("POST /chat", h.handleChat)
	mux.ServeHTTP(w, r)
}

// Handler returns an http.Handler mounted at the configured base path.
func (h *PlaygroundHandler) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("GET %s", h.opts.path), h.handleUI)
	mux.HandleFunc(fmt.Sprintf("GET %s/agents", h.opts.path), h.handleListAgents)
	mux.HandleFunc(fmt.Sprintf("POST %s/chat", h.opts.path), h.handleChat)
	return mux
}

func (h *PlaygroundHandler) handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, playgroundHTML, h.opts.title, h.opts.path)
}

func (h *PlaygroundHandler) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents := h.selector.List(r.Context())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"agents": agents})
}

type chatRequest struct {
	AgentID string `json:"agent_id"`
	Input   string `json:"input"`
}

func (h *PlaygroundHandler) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.AgentID == "" || req.Input == "" {
		http.Error(w, "agent_id and input are required", http.StatusBadRequest)
		return
	}

	a := h.selector.Get(r.Context(), req.AgentID)
	if a == nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	if err := h.adapter.WriteEvents(r.Context(), w, a, req.Input); err != nil {
		// Error may already be partially written to SSE stream.
		return
	}
}

// defaultStreamAdapter is the built-in SSE stream adapter.
type defaultStreamAdapter struct{}

var _ StreamAdapter = (*defaultStreamAdapter)(nil)

// WriteEvents streams agent events as SSE.
func (a *defaultStreamAdapter) WriteEvents(ctx context.Context, w http.ResponseWriter, ag agent.Agent, input string) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return fmt.Errorf("response writer does not support flushing")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for evt, err := range ag.Stream(ctx, input) {
		if err != nil {
			data, _ := json.Marshal(map[string]string{"type": "error", "text": err.Error()})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			return err
		}

		data, _ := json.Marshal(map[string]any{
			"type":     string(evt.Type),
			"text":     evt.Text,
			"agent_id": evt.AgentID,
		})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: {\"type\":\"done\"}\n\n")
	flusher.Flush()
	return nil
}

// StaticSelector is a simple AgentSelector backed by a map.
type StaticSelector struct {
	mu     sync.RWMutex
	agents map[string]agent.Agent
}

var _ AgentSelector = (*StaticSelector)(nil)

// NewStaticSelector creates a StaticSelector from the given agents.
func NewStaticSelector(agents ...agent.Agent) *StaticSelector {
	m := make(map[string]agent.Agent, len(agents))
	for _, a := range agents {
		m[a.ID()] = a
	}
	return &StaticSelector{agents: m}
}

// List returns all agent IDs sorted alphabetically.
func (s *StaticSelector) List(_ context.Context) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.agents))
	for id := range s.agents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Get returns the agent with the given ID, or nil.
func (s *StaticSelector) Get(_ context.Context, id string) agent.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agents[id]
}

const playgroundHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: system-ui, -apple-system, sans-serif; height: 100vh; display: flex; flex-direction: column; background: #f9fafb; }
  header { padding: 1rem; background: #1e293b; color: white; }
  header h1 { font-size: 1.25rem; }
  .container { flex: 1; display: flex; flex-direction: column; max-width: 800px; width: 100%%; margin: 0 auto; padding: 1rem; }
  .agent-select { margin-bottom: 1rem; }
  .agent-select select { padding: 0.5rem; border-radius: 0.375rem; border: 1px solid #d1d5db; width: 100%%; }
  .messages { flex: 1; overflow-y: auto; border: 1px solid #e5e7eb; border-radius: 0.5rem; padding: 1rem; background: white; margin-bottom: 1rem; }
  .message { margin-bottom: 0.75rem; padding: 0.75rem; border-radius: 0.5rem; max-width: 80%%; }
  .message.user { background: #dbeafe; margin-left: auto; }
  .message.assistant { background: #f3f4f6; }
  .input-row { display: flex; gap: 0.5rem; }
  .input-row input { flex: 1; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 0.375rem; }
  .input-row button { padding: 0.75rem 1.5rem; background: #2563eb; color: white; border: none; border-radius: 0.375rem; cursor: pointer; }
  .input-row button:hover { background: #1d4ed8; }
</style>
</head>
<body>
<header><h1>Beluga AI Playground</h1></header>
<div class="container">
  <div class="agent-select"><select id="agent"></select></div>
  <div class="messages" id="messages"></div>
  <div class="input-row">
    <input type="text" id="input" placeholder="Type a message..." />
    <button onclick="send()">Send</button>
  </div>
</div>
<script>
const basePath = "%s";
const agentSelect = document.getElementById("agent");
const messagesDiv = document.getElementById("messages");
const inputEl = document.getElementById("input");

fetch(basePath + "/agents").then(r => r.json()).then(data => {
  (data.agents || []).forEach(id => {
    const opt = document.createElement("option");
    opt.value = id; opt.textContent = id;
    agentSelect.appendChild(opt);
  });
});

inputEl.addEventListener("keydown", e => { if (e.key === "Enter") send(); });

function addMessage(role, text) {
  const div = document.createElement("div");
  div.className = "message " + role;
  div.textContent = text;
  messagesDiv.appendChild(div);
  messagesDiv.scrollTop = messagesDiv.scrollHeight;
  return div;
}

function send() {
  const input = inputEl.value.trim();
  if (!input) return;
  inputEl.value = "";
  addMessage("user", input);
  const assistantDiv = addMessage("assistant", "");

  fetch(basePath + "/chat", {
    method: "POST",
    headers: {"Content-Type": "application/json"},
    body: JSON.stringify({agent_id: agentSelect.value, input: input})
  }).then(response => {
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    function read() {
      reader.read().then(({done, value}) => {
        if (done) return;
        buffer += decoder.decode(value, {stream: true});
        const lines = buffer.split("\n");
        buffer = lines.pop();
        lines.forEach(line => {
          if (line.startsWith("data: ")) {
            try {
              const evt = JSON.parse(line.slice(6));
              if (evt.text) assistantDiv.textContent += evt.text;
            } catch(e) {}
          }
        });
        read();
      });
    }
    read();
  });
}
</script>
</body>
</html>`
