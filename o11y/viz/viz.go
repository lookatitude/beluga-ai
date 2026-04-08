package viz

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// GraphRenderer renders an ExecutionGraph to a specific format.
type GraphRenderer interface {
	// Render converts the graph into a byte representation (JSON, DOT, etc.).
	Render(ctx context.Context, graph *ExecutionGraph) ([]byte, error)
}

// EventCollector collects execution events and builds an ExecutionGraph.
type EventCollector interface {
	// Record adds an execution event to the collector.
	Record(ctx context.Context, event ExecutionEvent) error
	// Build constructs an ExecutionGraph from all collected events.
	Build(ctx context.Context) (*ExecutionGraph, error)
}

// ExecutionGraph represents the full execution trace of an agent as a directed graph.
type ExecutionGraph struct {
	// ID uniquely identifies this execution graph.
	ID string `json:"id"`
	// RootNodeID is the entry point of the execution.
	RootNodeID string `json:"root_node_id"`
	// Nodes are the execution steps.
	Nodes []Node `json:"nodes"`
	// Edges connect nodes in execution order.
	Edges []Edge `json:"edges"`
	// StartTime is when execution began.
	StartTime time.Time `json:"start_time"`
	// EndTime is when execution completed.
	EndTime time.Time `json:"end_time"`
}

// Node represents a single execution step in the graph.
type Node struct {
	// ID uniquely identifies this node.
	ID string `json:"id"`
	// Label is a human-readable description.
	Label string `json:"label"`
	// Type classifies the node (e.g., "agent", "tool", "llm", "handoff").
	Type string `json:"type"`
	// AgentID identifies which agent owns this step.
	AgentID string `json:"agent_id,omitempty"`
	// StartTime is when this step began.
	StartTime time.Time `json:"start_time"`
	// Duration is how long this step took.
	Duration time.Duration `json:"duration"`
	// Status is the outcome: "success", "error", "skipped".
	Status string `json:"status"`
	// Metadata holds extra attributes.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Edge represents a directed connection between two nodes.
type Edge struct {
	// From is the source node ID.
	From string `json:"from"`
	// To is the target node ID.
	To string `json:"to"`
	// Label describes the relationship.
	Label string `json:"label,omitempty"`
}

// ExecutionEvent is a raw event captured during agent execution.
type ExecutionEvent struct {
	// ID uniquely identifies this event.
	ID string
	// ParentID links to the parent event (empty for root).
	ParentID string
	// Type classifies the event.
	Type string
	// AgentID identifies which agent produced this event.
	AgentID string
	// Label is a human-readable description.
	Label string
	// Timestamp is when this event occurred.
	Timestamp time.Time
	// Duration is how long the operation took (zero if still in progress).
	Duration time.Duration
	// Status is the outcome.
	Status string
	// Metadata holds extra attributes.
	Metadata map[string]any
}

// Option configures a DefaultCollector.
type Option func(*collectorOptions)

type collectorOptions struct {
	graphID string
}

// WithGraphID sets the graph ID for the collector.
func WithGraphID(id string) Option {
	return func(o *collectorOptions) { o.graphID = id }
}

// DefaultCollector is the standard EventCollector implementation.
type DefaultCollector struct {
	mu     sync.Mutex
	events []ExecutionEvent
	opts   collectorOptions
}

var _ EventCollector = (*DefaultCollector)(nil)

// NewCollector creates a new DefaultCollector.
func NewCollector(opts ...Option) *DefaultCollector {
	o := collectorOptions{graphID: "exec-graph"}
	for _, opt := range opts {
		opt(&o)
	}
	return &DefaultCollector{opts: o}
}

// Record adds an event to the collector. It is safe for concurrent use.
func (c *DefaultCollector) Record(_ context.Context, event ExecutionEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
	return nil
}

// Build constructs an ExecutionGraph from the collected events.
func (c *DefaultCollector) Build(_ context.Context) (*ExecutionGraph, error) {
	c.mu.Lock()
	events := make([]ExecutionEvent, len(c.events))
	copy(events, c.events)
	c.mu.Unlock()

	if len(events) == 0 {
		return &ExecutionGraph{ID: c.opts.graphID}, nil
	}

	graph := &ExecutionGraph{
		ID:        c.opts.graphID,
		StartTime: events[0].Timestamp,
		EndTime:   events[len(events)-1].Timestamp,
	}

	for _, e := range events {
		node := Node{
			ID:        e.ID,
			Label:     e.Label,
			Type:      e.Type,
			AgentID:   e.AgentID,
			StartTime: e.Timestamp,
			Duration:  e.Duration,
			Status:    e.Status,
			Metadata:  e.Metadata,
		}
		graph.Nodes = append(graph.Nodes, node)

		if e.ParentID != "" {
			graph.Edges = append(graph.Edges, Edge{
				From: e.ParentID,
				To:   e.ID,
			})
		} else if graph.RootNodeID == "" {
			graph.RootNodeID = e.ID
		}
	}

	return graph, nil
}

// JSONRenderer renders an ExecutionGraph as JSON.
type JSONRenderer struct{}

var _ GraphRenderer = (*JSONRenderer)(nil)

// Render produces a JSON representation of the graph.
func (r *JSONRenderer) Render(_ context.Context, graph *ExecutionGraph) ([]byte, error) {
	return json.MarshalIndent(graph, "", "  ")
}

// DOTRenderer renders an ExecutionGraph in Graphviz DOT format.
type DOTRenderer struct{}

var _ GraphRenderer = (*DOTRenderer)(nil)

// Render produces a DOT representation of the graph.
func (r *DOTRenderer) Render(_ context.Context, graph *ExecutionGraph) ([]byte, error) {
	var b strings.Builder
	b.WriteString("digraph execution {\n")
	b.WriteString("  rankdir=TB;\n")
	b.WriteString("  node [shape=box, style=rounded];\n\n")

	// Sort nodes for deterministic output.
	nodes := make([]Node, len(graph.Nodes))
	copy(nodes, graph.Nodes)
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	for _, n := range nodes {
		color := nodeColor(n.Status)
		label := fmt.Sprintf("%s\\n[%s] %s", n.Label, n.Type, n.Duration.Round(time.Millisecond))
		b.WriteString(fmt.Sprintf("  %q [label=%q, color=%q];\n", n.ID, label, color))
	}

	b.WriteString("\n")

	edges := make([]Edge, len(graph.Edges))
	copy(edges, graph.Edges)
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})

	for _, e := range edges {
		if e.Label != "" {
			b.WriteString(fmt.Sprintf("  %q -> %q [label=%q];\n", e.From, e.To, e.Label))
		} else {
			b.WriteString(fmt.Sprintf("  %q -> %q;\n", e.From, e.To))
		}
	}

	b.WriteString("}\n")
	return []byte(b.String()), nil
}

func nodeColor(status string) string {
	switch status {
	case "success":
		return "green"
	case "error":
		return "red"
	case "skipped":
		return "gray"
	default:
		return "black"
	}
}

// Factory creates a GraphRenderer.
type Factory func() (GraphRenderer, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a renderer factory to the global registry.
func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

// NewRenderer creates a GraphRenderer by name from the registry.
func NewRenderer(name string) (GraphRenderer, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("viz: unknown renderer %q (registered: %v)", name, ListRenderers())
	}
	return f()
}

// ListRenderers returns the names of all registered renderers, sorted alphabetically.
func ListRenderers() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func init() {
	Register("json", func() (GraphRenderer, error) { return &JSONRenderer{}, nil })
	Register("dot", func() (GraphRenderer, error) { return &DOTRenderer{}, nil })
}
