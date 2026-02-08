package orchestration

import (
	"context"
	"fmt"
	"iter"

	"github.com/lookatitude/beluga-ai/core"
)

// Edge represents a directed connection between two nodes in a Graph.
// If Condition is nil, the edge is unconditional (always taken).
type Edge struct {
	From      string
	To        string
	Condition func(any) bool // nil = unconditional
}

// Graph is a directed graph of named Runnable nodes connected by conditional
// edges. Traversal starts at the entry node and follows matching edges until
// a terminal node (no outgoing edges or no matching conditions) is reached.
//
// For multiple matching edges from a node, the first match wins.
type Graph struct {
	nodes map[string]core.Runnable
	edges []Edge
	entry string
}

// NewGraph creates an empty Graph.
func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]core.Runnable),
	}
}

// AddNode registers a named node in the graph.
func (g *Graph) AddNode(name string, r core.Runnable) error {
	if name == "" {
		return fmt.Errorf("orchestration/graph: node name must not be empty")
	}
	if _, exists := g.nodes[name]; exists {
		return fmt.Errorf("orchestration/graph: duplicate node %q", name)
	}
	g.nodes[name] = r
	return nil
}

// AddEdge adds a directed edge to the graph.
func (g *Graph) AddEdge(edge Edge) error {
	if _, ok := g.nodes[edge.From]; !ok {
		return fmt.Errorf("orchestration/graph: unknown source node %q", edge.From)
	}
	if _, ok := g.nodes[edge.To]; !ok {
		return fmt.Errorf("orchestration/graph: unknown target node %q", edge.To)
	}
	g.edges = append(g.edges, edge)
	return nil
}

// SetEntry sets the entry node for graph traversal.
func (g *Graph) SetEntry(name string) error {
	if _, ok := g.nodes[name]; !ok {
		return fmt.Errorf("orchestration/graph: unknown entry node %q", name)
	}
	g.entry = name
	return nil
}

// maxTraversalDepth limits graph traversal to prevent infinite loops.
const maxTraversalDepth = 100

// Invoke traverses the graph starting from the entry node.
func (g *Graph) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	if g.entry == "" {
		return nil, fmt.Errorf("orchestration/graph: no entry node set")
	}

	current := g.entry
	value := input

	for depth := 0; depth < maxTraversalDepth; depth++ {
		node, ok := g.nodes[current]
		if !ok {
			return nil, fmt.Errorf("orchestration/graph: node %q not found", current)
		}

		result, err := node.Invoke(ctx, value, opts...)
		if err != nil {
			return nil, fmt.Errorf("orchestration/graph: node %q: %w", current, err)
		}
		value = result

		// Find next matching edge.
		next := g.nextNode(current, value)
		if next == "" {
			// Terminal node — no outgoing edges or no matching conditions.
			return value, nil
		}
		current = next
	}

	return nil, fmt.Errorf("orchestration/graph: max traversal depth (%d) exceeded", maxTraversalDepth)
}

// Stream traverses the graph, streaming the last node's output.
func (g *Graph) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		if g.entry == "" {
			yield(nil, fmt.Errorf("orchestration/graph: no entry node set"))
			return
		}

		current := g.entry
		value := input

		for depth := 0; depth < maxTraversalDepth; depth++ {
			node, ok := g.nodes[current]
			if !ok {
				yield(nil, fmt.Errorf("orchestration/graph: node %q not found", current))
				return
			}

			// Check if this is a terminal node (no matching next).
			// We need to invoke first to get the value for edge conditions,
			// but for the terminal node we want to stream instead.
			// Strategy: peek at edges. If no outgoing edges, stream this node.
			if !g.hasOutgoingEdges(current) {
				for val, err := range node.Stream(ctx, value, opts...) {
					if !yield(val, err) {
						return
					}
					if err != nil {
						return
					}
				}
				return
			}

			// Non-terminal: invoke and follow edges.
			result, err := node.Invoke(ctx, value, opts...)
			if err != nil {
				yield(nil, fmt.Errorf("orchestration/graph: node %q: %w", current, err))
				return
			}
			value = result

			next := g.nextNode(current, value)
			if next == "" {
				// No matching conditions — this is effectively terminal.
				yield(value, nil)
				return
			}
			current = next
		}

		yield(nil, fmt.Errorf("orchestration/graph: max traversal depth (%d) exceeded", maxTraversalDepth))
	}
}

// nextNode returns the name of the next node to visit from the given node,
// or "" if no outgoing edge matches.
func (g *Graph) nextNode(from string, value any) string {
	for _, e := range g.edges {
		if e.From != from {
			continue
		}
		if e.Condition == nil || e.Condition(value) {
			return e.To
		}
	}
	return ""
}

// hasOutgoingEdges returns true if the node has any outgoing edges defined.
func (g *Graph) hasOutgoingEdges(name string) bool {
	for _, e := range g.edges {
		if e.From == name {
			return true
		}
	}
	return false
}
