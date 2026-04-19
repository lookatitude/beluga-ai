package agui

import (
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
)

// AGUIHandler serves agents via the AG-UI protocol, providing a standardized
// interface for UI clients to discover and interact with agents.
type AGUIHandler interface {
	// ServeHTTP handles AG-UI protocol requests.
	http.Handler
}

// AgentsManifest represents the AGENTS.md machine-readable manifest
// describing available agents and their capabilities.
type AgentsManifest struct {
	// Version is the manifest format version.
	Version string `json:"version"`
	// Agents lists all available agents.
	Agents []AgentEntry `json:"agents"`
	// GeneratedAt is when this manifest was generated.
	GeneratedAt time.Time `json:"generated_at"`
}

// AgentEntry describes a single agent in the manifest.
type AgentEntry struct {
	// ID is the unique agent identifier.
	ID string `json:"id"`
	// Name is a human-readable name.
	Name string `json:"name"`
	// Description explains what the agent does.
	Description string `json:"description,omitempty"`
	// Capabilities lists what the agent can do.
	Capabilities []string `json:"capabilities,omitempty"`
	// Endpoint is the URL for interacting with this agent.
	Endpoint string `json:"endpoint,omitempty"`
	// Tools lists the tool names available to this agent.
	Tools []string `json:"tools,omitempty"`
}

// UIEvent is an event sent from agent to UI via the AG-UI protocol.
type UIEvent struct {
	// Type identifies the event kind.
	Type string `json:"type"`
	// AgentID identifies the source agent.
	AgentID string `json:"agent_id"`
	// Data holds the event payload.
	Data any `json:"data,omitempty"`
	// Timestamp is when this event was created.
	Timestamp time.Time `json:"timestamp"`
}

// Option configures an AGUIHandler.
type Option func(*handlerOptions)

type handlerOptions struct {
	basePath string
	version  string
}

// WithBasePath sets the base URL path for the handler.
func WithBasePath(p string) Option {
	return func(o *handlerOptions) { o.basePath = p }
}

// WithVersion sets the manifest version string.
func WithVersion(v string) Option {
	return func(o *handlerOptions) { o.version = v }
}

// DefaultHandler is the standard AG-UI handler implementation.
type DefaultHandler struct {
	mu     sync.RWMutex
	agents map[string]agent.Agent
	opts   handlerOptions
}

var _ AGUIHandler = (*DefaultHandler)(nil)

// NewHandler creates a new AG-UI handler for the given agents.
func NewHandler(agents []agent.Agent, opts ...Option) *DefaultHandler {
	o := handlerOptions{
		basePath: "/agui",
		version:  "1.0",
	}
	for _, opt := range opts {
		opt(&o)
	}

	m := make(map[string]agent.Agent, len(agents))
	for _, a := range agents {
		m[a.ID()] = a
	}

	return &DefaultHandler{agents: m, opts: o}
}

// ServeHTTP routes AG-UI protocol requests.
func (h *DefaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == h.opts.basePath+"/agents" || path == h.opts.basePath+"/agents/":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleAgents(w, r)
	case path == h.opts.basePath+"/manifest" || path == h.opts.basePath+"/manifest/":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleManifest(w, r)
	case strings.HasPrefix(path, h.opts.basePath+"/chat/"):
		h.handleChat(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *DefaultHandler) handleAgents(w http.ResponseWriter, _ *http.Request) {
	manifest := h.buildManifest()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(manifest.Agents); err != nil {
		slog.Error("agui: encode agents", "err", err)
	}
}

func (h *DefaultHandler) handleManifest(w http.ResponseWriter, _ *http.Request) {
	manifest := h.buildManifest()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(manifest); err != nil {
		slog.Error("agui: encode manifest", "err", err)
	}
}

func (h *DefaultHandler) handleChat(w http.ResponseWriter, r *http.Request) {
	// Method check before agent lookup: avoids leaking whether an agent
	// exists via the 405 vs 404 response difference.
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from path: /agui/chat/{agentID}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, h.opts.basePath+"/chat/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "agent ID required", http.StatusBadRequest)
		return
	}
	agentID := parts[0]

	h.mu.RLock()
	a, ok := h.agents[agentID]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	var req struct {
		Input string `json:"input"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Sanitize the agent ID before placing it in log fields to prevent
	// log injection via attacker-controlled path segments.
	safeAgentID := sanitizeLogValue(agentID)

	flusher, ok := w.(http.Flusher)
	if !ok {
		// Fall back to non-streaming.
		result, err := a.Invoke(r.Context(), req.Input)
		if err != nil {
			// Do not leak provider error details to clients; log server-side.
			slog.Error("agui: agent invoke failed", "agent_id", safeAgentID, "err", sanitizeLogValue(err.Error()))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"result": result}); err != nil {
			slog.Error("agui: encode chat response", "agent_id", safeAgentID, "err", sanitizeLogValue(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	for evt, err := range a.Stream(r.Context(), req.Input) {
		if err != nil {
			// Log detailed error server-side; send sanitized message to client.
			slog.Error("agui: agent stream failed", "agent_id", safeAgentID, "err", sanitizeLogValue(err.Error()))
			data, _ := json.Marshal(UIEvent{Type: "error", AgentID: agentID, Data: "internal server error", Timestamp: time.Now()})
			if _, werr := fmt.Fprintf(w, "data: %s\n\n", data); werr != nil {
				slog.Error("agui: sse write failed", "agent_id", safeAgentID, "err", sanitizeLogValue(werr.Error()))
				return
			}
			flusher.Flush()
			return
		}
		data, _ := json.Marshal(UIEvent{
			Type:      string(evt.Type),
			AgentID:   agentID,
			Data:      evt.Text,
			Timestamp: time.Now(),
		})
		if _, werr := fmt.Fprintf(w, "data: %s\n\n", data); werr != nil {
			slog.Error("agui: sse write failed", "agent_id", safeAgentID, "err", sanitizeLogValue(werr.Error()))
			return
		}
		flusher.Flush()
	}
}

// sanitizeLogValue removes CR/LF and other control characters from a string
// before inserting it into a structured log field. This prevents log injection
// when attacker-controlled data (URL path segments, provider errors) flows
// into log records.
func sanitizeLogValue(s string) string {
	if len(s) > 256 {
		s = s[:256]
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteByte(' ')
			continue
		}
		if r < 0x20 || r == 0x7f {
			b.WriteByte('?')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (h *DefaultHandler) buildManifest() *AgentsManifest {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entries := make([]AgentEntry, 0, len(h.agents))
	for _, a := range h.agents {
		var toolNames []string
		for _, t := range a.Tools() {
			toolNames = append(toolNames, t.Name())
		}

		entries = append(entries, AgentEntry{
			ID:       a.ID(),
			Name:     a.Persona().Role,
			Endpoint: h.opts.basePath + "/chat/" + a.ID(),
			Tools:    toolNames,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })

	return &AgentsManifest{
		Version:     h.opts.version,
		Agents:      entries,
		GeneratedAt: time.Now(),
	}
}

// ParseManifest parses an AgentsManifest from JSON bytes.
func ParseManifest(data []byte) (*AgentsManifest, error) {
	var m AgentsManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("agui: parse manifest: %w", err)
	}
	return &m, nil
}

// GenerateManifest creates a JSON representation of the agents manifest.
func GenerateManifest(agents []agent.Agent, opts ...Option) ([]byte, error) {
	h := NewHandler(agents, opts...)
	manifest := h.buildManifest()
	return json.MarshalIndent(manifest, "", "  ")
}

// GenerateMarkdown creates an AGENTS.md markdown document from agents.
func GenerateMarkdown(agents []agent.Agent) string {
	var b strings.Builder
	b.WriteString("# AGENTS.md\n\n")
	b.WriteString("Machine-readable agent manifest for discovery.\n\n")

	sorted := make([]agent.Agent, len(agents))
	copy(sorted, agents)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID() < sorted[j].ID() })

	for _, a := range sorted {
		b.WriteString(fmt.Sprintf("## %s\n\n", a.ID()))
		b.WriteString(fmt.Sprintf("- **Role**: %s\n", a.Persona().Role))
		if a.Persona().Goal != "" {
			b.WriteString(fmt.Sprintf("- **Goal**: %s\n", a.Persona().Goal))
		}
		tools := a.Tools()
		if len(tools) > 0 {
			b.WriteString("- **Tools**: ")
			for i, t := range tools {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(t.Name())
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// StreamToUIEvents converts an agent event stream to UIEvent stream.
func StreamToUIEvents(agentID string, stream iter.Seq2[agent.Event, error]) iter.Seq2[UIEvent, error] {
	return func(yield func(UIEvent, error) bool) {
		for evt, err := range stream {
			if err != nil {
				if !yield(UIEvent{}, err) {
					return
				}
				continue
			}
			uiEvt := UIEvent{
				Type:      string(evt.Type),
				AgentID:   agentID,
				Data:      evt.Text,
				Timestamp: time.Now(),
			}
			if !yield(uiEvt, nil) {
				return
			}
		}
	}
}
